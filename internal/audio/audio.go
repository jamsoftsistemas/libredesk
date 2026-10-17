// Package audio converts audio blobs between formats using the ffmpeg binary.
package audio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// convertTimeout bounds a single ffmpeg run.
const convertTimeout = 2 * time.Minute

// ErrFFmpegNotFound is returned when the ffmpeg binary is not on PATH.
var ErrFFmpegNotFound = errors.New("ffmpeg not found in PATH")

var (
	ffmpegOnce sync.Once
	ffmpegPath string
)

// Available reports whether ffmpeg is installed.
func Available() bool {
	ffmpegOnce.Do(func() {
		ffmpegPath, _ = exec.LookPath("ffmpeg")
	})
	return ffmpegPath != ""
}

// baseType strips parameters and lowercases a content type.
func baseType(contentType string) string {
	ct := strings.ToLower(strings.TrimSpace(contentType))
	if i := strings.IndexByte(ct, ';'); i >= 0 {
		ct = strings.TrimSpace(ct[:i])
	}
	return ct
}

// IsAudio reports whether the content type is audio/*.
func IsAudio(contentType string) bool {
	return strings.HasPrefix(baseType(contentType), "audio/")
}

// NeedsConversion reports whether an audio content type should be converted to ogg/opus
// so every channel (WhatsApp included) and the transcription provider accept it.
func NeedsConversion(contentType string) bool {
	if !IsAudio(contentType) {
		return false
	}
	switch baseType(contentType) {
	case "audio/ogg", "audio/opus", "audio/mpeg", "audio/mp3", "audio/mp4", "audio/aac", "audio/amr":
		return false
	}
	return true
}

// ConvertToOggOpus converts any audio blob to ogg/opus (WhatsApp voice note format).
func ConvertToOggOpus(ctx context.Context, data []byte) ([]byte, error) {
	return run(ctx, data, "-vn", "-c:a", "libopus", "-b:a", "32k", "-ac", "1", "-f", "ogg")
}

// ConvertToMP3 converts any audio blob to mp3.
func ConvertToMP3(ctx context.Context, data []byte) ([]byte, error) {
	return run(ctx, data, "-vn", "-c:a", "libmp3lame", "-b:a", "64k", "-ac", "1", "-f", "mp3")
}

func run(ctx context.Context, data []byte, outArgs ...string) ([]byte, error) {
	if !Available() {
		return nil, ErrFFmpegNotFound
	}
	ctx, cancel := context.WithTimeout(ctx, convertTimeout)
	defer cancel()

	args := append([]string{"-hide_banner", "-loglevel", "error", "-i", "pipe:0"}, outArgs...)
	args = append(args, "pipe:1")
	cmd := exec.CommandContext(ctx, ffmpegPath, args...)
	cmd.Stdin = bytes.NewReader(data)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	if stdout.Len() == 0 {
		return nil, errors.New("ffmpeg produced empty output")
	}
	return stdout.Bytes(), nil
}
