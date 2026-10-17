package main

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai"
	"github.com/abhinavxd/libredesk/internal/audio"
	"github.com/abhinavxd/libredesk/internal/conversation"
)

// convertUploadedAudio converts browser-recorded audio (e.g. webm) to ogg/opus so every channel accepts it
// and WhatsApp renders it as a voice note. On failure or when no conversion is needed the input is returned as is.
func convertUploadedAudio(app *App, data []byte, fileName, contentType string) ([]byte, string, string, bool) {
	if !audio.NeedsConversion(contentType) {
		return data, fileName, contentType, false
	}
	converted, err := audio.ConvertToOggOpus(context.Background(), data)
	if err != nil {
		app.lo.Warn("could not convert uploaded audio to ogg, keeping original", "content_type", contentType, "error", err)
		return data, fileName, contentType, false
	}
	name := strings.TrimSuffix(fileName, filepath.Ext(fileName)) + ".ogg"
	return converted, name, "audio/ogg", true
}

// transcriberAdapter bridges the AI manager to the conversation manager's Transcriber.
type transcriberAdapter struct{ ai *ai.Manager }

func (t transcriberAdapter) TranscribeAudio(ctx context.Context, data []byte, contentType string) (string, error) {
	text, err := t.ai.TranscribeAudio(ctx, data, contentType)
	if errors.Is(err, ai.ErrTranscriptionDisabled) {
		return "", conversation.ErrTranscriptionDisabled
	}
	return text, err
}
