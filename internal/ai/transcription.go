package ai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/audio"
)

const (
	defaultTranscriptionModel = "google/gemini-2.5-flash"

	transcriptionPrompt = "Transcribe this audio verbatim, in the language that is spoken. Reply only with the transcription, without any preamble, quotes or comments. If there is no intelligible speech, reply with an empty string."
)

// ErrTranscriptionDisabled is returned when no transcription provider API key is configured.
var ErrTranscriptionDisabled = errors.New("transcription provider not configured")

// TranscribeAudio transcribes an audio blob with the configured transcription provider.
func (m *Manager) TranscribeAudio(ctx context.Context, data []byte, contentType string) (string, error) {
	cfg, err := m.getProviderConfig(models.ProviderTypeTranscription)
	if err != nil {
		return "", err
	}
	if cfg.APIKey == "" {
		return "", ErrTranscriptionDisabled
	}

	format, ok := transcriptionFormat(contentType)
	if !ok {
		converted, cerr := audio.ConvertToMP3(ctx, data)
		if cerr != nil {
			return "", fmt.Errorf("converting %s audio to mp3: %w", contentType, cerr)
		}
		data, format = converted, "mp3"
	}

	client := NewOpenAIClient(cfg, m.lo, m.providerHTTPClient)
	return client.TranscribeAudio(ctx, data, format)
}

// transcriptionFormat maps a content type to an OpenRouter input_audio format.
func transcriptionFormat(contentType string) (string, bool) {
	ct := strings.ToLower(contentType)
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	switch ct {
	case "audio/ogg", "audio/opus":
		return "ogg", true
	case "audio/mpeg", "audio/mp3":
		return "mp3", true
	case "audio/wav", "audio/x-wav", "audio/wave":
		return "wav", true
	case "audio/mp4", "audio/m4a", "audio/x-m4a":
		return "m4a", true
	case "audio/aac":
		return "aac", true
	case "audio/flac", "audio/x-flac":
		return "flac", true
	case "audio/aiff", "audio/x-aiff":
		return "aiff", true
	}
	return "", false
}

// TranscribeAudio sends the audio as an input_audio chat content part (OpenRouter format) and returns the transcript.
func (o *OpenAIClient) TranscribeAudio(ctx context.Context, data []byte, format string) (string, error) {
	if o.cfg.APIKey == "" {
		return "", ErrApiKeyNotSet
	}
	model := o.cfg.Model
	if model == "" {
		model = defaultTranscriptionModel
	}
	body := map[string]any{
		"model": model,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": transcriptionPrompt},
					{"type": "input_audio", "input_audio": map[string]string{
						"data":   base64.StdEncoding.EncodeToString(data),
						"format": format,
					}},
				},
			},
		},
	}
	if o.cfg.Temperature != nil {
		body["temperature"] = *o.cfg.Temperature
	}

	respBytes, err := o.post(ctx, "/chat/completions", body)
	if err != nil {
		return "", err
	}
	var parsed chatCompletionResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("decoding response body: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("no response found")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}
