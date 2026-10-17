package ai

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/abhinavxd/libredesk/internal/ai/models"
	"github.com/abhinavxd/libredesk/internal/audio"
)

const defaultTranscriptionModel = "openai/whisper-large-v3"

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
	case "audio/webm":
		return "webm", true
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

// transcriptionResponse is the /audio/transcriptions response body.
type transcriptionResponse struct {
	Text string `json:"text"`
}

// TranscribeAudio posts base64 audio to the OpenRouter-style /audio/transcriptions endpoint and returns the text.
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
		"input_audio": map[string]string{
			"data":   base64.StdEncoding.EncodeToString(data),
			"format": format,
		},
	}
	if o.cfg.Temperature != nil {
		body["temperature"] = *o.cfg.Temperature
	}

	respBytes, err := o.post(ctx, "/audio/transcriptions", body)
	if err != nil {
		return "", err
	}
	var parsed transcriptionResponse
	if err := json.Unmarshal(respBytes, &parsed); err != nil {
		return "", fmt.Errorf("decoding response body: %w", err)
	}
	return strings.TrimSpace(parsed.Text), nil
}

// testAudioWAV returns half a second of 16 kHz mono silence as a WAV file, used to test the transcription provider.
func testAudioWAV() []byte {
	const (
		sampleRate = 16000
		samples    = sampleRate / 2
		dataLen    = samples * 2
	)
	b := make([]byte, 44+dataLen)
	copy(b[0:], "RIFF")
	binary.LittleEndian.PutUint32(b[4:], uint32(36+dataLen))
	copy(b[8:], "WAVEfmt ")
	binary.LittleEndian.PutUint32(b[16:], 16)
	binary.LittleEndian.PutUint16(b[20:], 1) // PCM
	binary.LittleEndian.PutUint16(b[22:], 1) // mono
	binary.LittleEndian.PutUint32(b[24:], sampleRate)
	binary.LittleEndian.PutUint32(b[28:], sampleRate*2)
	binary.LittleEndian.PutUint16(b[32:], 2)
	binary.LittleEndian.PutUint16(b[34:], 16)
	copy(b[36:], "data")
	binary.LittleEndian.PutUint32(b[40:], dataLen)
	return b
}
