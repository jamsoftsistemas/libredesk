package conversation

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
)

const (
	transcriptionQueueSize = 1000

	// TranscriptionTimeout bounds one transcription attempt, and also doubles as the window a reader
	// (e.g. the AI agent, or the frontend) should wait on an audio message before giving up on ever
	// seeing a transcription for it.
	TranscriptionTimeout = 3 * time.Minute

	TranscriptionStatusDone   = "done"
	TranscriptionStatusFailed = "failed"
)

// ErrTranscriptionDisabled is returned by a Transcriber when no provider is configured; the job is dropped silently.
var ErrTranscriptionDisabled = errors.New("transcription disabled")

// Transcriber turns an audio blob into text.
type Transcriber interface {
	TranscribeAudio(ctx context.Context, data []byte, contentType string) (string, error)
}

type transcriptionJob struct {
	MessageUUID string
	MediaUUID   string
	ContentType string
}

// SetTranscriber sets the audio transcriber; when nil, audio attachments are not transcribed.
func (m *Manager) SetTranscriber(t Transcriber) {
	m.transcriber = t
}

// RunTranscriptionWorkers starts workers that transcribe audio attachments until ctx is done.
func (m *Manager) RunTranscriptionWorkers(ctx context.Context, workers int) {
	for range workers {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-m.transcriptionQueue:
					m.transcribe(ctx, job)
				}
			}
		}()
	}
}

// enqueueTranscriptions queues every audio attachment of the message for transcription.
func (m *Manager) enqueueTranscriptions(message *models.Message) {
	if m.transcriber == nil {
		return
	}
	for _, att := range message.Attachments {
		if att.UUID == "" || !strings.HasPrefix(strings.ToLower(att.ContentType), "audio/") {
			continue
		}
		select {
		case m.transcriptionQueue <- transcriptionJob{MessageUUID: message.UUID, MediaUUID: att.UUID, ContentType: att.ContentType}:
		default:
			m.lo.Warn("transcription queue full, skipping audio", "message_uuid", message.UUID, "media_uuid", att.UUID)
		}
	}
}

func (m *Manager) transcribe(ctx context.Context, job transcriptionJob) {
	blob, err := m.mediaStore.GetBlob(job.MediaUUID)
	if err != nil {
		m.lo.Error("error fetching audio blob for transcription", "media_uuid", job.MediaUUID, "error", err)
		return
	}

	ctx, cancel := context.WithTimeout(ctx, TranscriptionTimeout)
	defer cancel()

	result := map[string]any{"updated_at": time.Now().UTC().Format(time.RFC3339)}
	text, err := m.transcriber.TranscribeAudio(ctx, blob, job.ContentType)
	switch {
	case errors.Is(err, ErrTranscriptionDisabled):
		return
	case err != nil:
		m.lo.Error("error transcribing audio", "message_uuid", job.MessageUUID, "media_uuid", job.MediaUUID, "error", err)
		result["status"] = TranscriptionStatusFailed
	default:
		result["status"] = TranscriptionStatusDone
		result["text"] = text
	}

	b, err := json.Marshal(result)
	if err != nil {
		return
	}
	conversationUUID, err := m.mergeTranscription(job.MessageUUID, job.MediaUUID, b)
	if err != nil {
		m.lo.Error("error saving transcription", "message_uuid", job.MessageUUID, "error", err)
		return
	}
	// The AI agent may have skipped replying while this transcription was in flight (see
	// AudioTranscriptionSettled); wake it up now that it has settled, done or failed, so the
	// customer isn't left waiting on an audio message until something else happens to retrigger it.
	m.notifyAIAgent(conversationUUID)
}

// SaveAudioTranscription persists a transcription produced outside the internal worker (e.g. one a
// channel provider like UAZAPI already transcribed) into message.Meta, in the same shape the
// internal transcription worker uses, so readers like AudioTranscription resolve it either way.
func (m *Manager) SaveAudioTranscription(messageUUID, mediaUUID, text string) error {
	b, err := json.Marshal(map[string]any{
		"status":     TranscriptionStatusDone,
		"text":       text,
		"updated_at": time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}
	conversationUUID, err := m.mergeTranscription(messageUUID, mediaUUID, b)
	if err != nil {
		return err
	}
	m.notifyAIAgent(conversationUUID)
	return nil
}

// mergeTranscription writes one attachment's transcription result into its message's meta and
// returns the owning conversation's UUID.
func (m *Manager) mergeTranscription(messageUUID, mediaUUID string, result []byte) (string, error) {
	var row struct {
		UUID             string          `db:"uuid"`
		ConversationUUID string          `db:"conversation_uuid"`
		Meta             json.RawMessage `db:"meta"`
	}
	if err := m.q.MergeMessageTranscription.Get(&row, messageUUID, mediaUUID, result); err != nil {
		return "", err
	}
	m.BroadcastMessageUpdate(row.ConversationUUID, row.UUID, map[string]any{"meta": stripCSATUUID(row.Meta)})
	return row.ConversationUUID, nil
}

// notifyAIAgent re-triggers the AI agent for a conversation it may have deferred responding to
// (e.g. while an audio transcription it was waiting on was still in flight). A no-op when the
// conversation isn't assigned to an AI assistant.
func (m *Manager) notifyAIAgent(conversationUUID string) {
	if m.aiAgent == nil || conversationUUID == "" {
		return
	}
	conv, err := m.GetConversation(0, conversationUUID, "")
	if err != nil {
		m.lo.Error("error loading conversation to notify ai agent of transcription", "conversation_uuid", conversationUUID, "error", err)
		return
	}
	if !conv.AssignedUserID.Valid {
		return
	}
	m.aiAgent.HandleConversationEvent(conv.ID, int(conv.AssignedUserID.Int))
}
