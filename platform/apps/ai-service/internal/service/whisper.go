package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

// TranscriptionProvider — интерфейс транскрипции, позволяет заменить backend.
type TranscriptionProvider interface {
	Transcribe(ctx context.Context, audioPath, language string) (string, error)
}

// WhisperTranscriber — реализация через OpenAI Whisper API.
type WhisperTranscriber struct {
	apiKey  string
	baseURL string
	client  *http.Client
	logger  *zap.Logger
}

func NewWhisperTranscriber(apiKey string, logger *zap.Logger) *WhisperTranscriber {
	return &WhisperTranscriber{
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1/audio/transcriptions",
		client:  &http.Client{Timeout: 10 * time.Minute},
		logger:  logger,
	}
}

// Transcribe отправляет аудиофайл в OpenAI Whisper и возвращает текст.
func (w *WhisperTranscriber) Transcribe(ctx context.Context, audioPath, language string) (string, error) {
	w.logger.Info("whisper: starting transcription", zap.String("path", audioPath), zap.String("lang", language))

	f, err := os.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("whisper: open file: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)

	fw, err := mw.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return "", fmt.Errorf("whisper: create form file: %w", err)
	}
	if _, err = io.Copy(fw, f); err != nil {
		return "", fmt.Errorf("whisper: copy file: %w", err)
	}

	_ = mw.WriteField("model", "whisper-1")
	_ = mw.WriteField("response_format", "text")

	if language != "" && language != "auto" {
		_ = mw.WriteField("language", language)
	}

	if err = mw.Close(); err != nil {
		return "", fmt.Errorf("whisper: close writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.baseURL, &buf)
	if err != nil {
		return "", fmt.Errorf("whisper: new request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+w.apiKey)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := w.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("whisper: do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("whisper: read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("whisper: api error %d: %s", resp.StatusCode, string(body))
	}

	transcript := string(body)
	w.logger.Info("whisper: transcription done", zap.Int("chars", len(transcript)))
	return transcript, nil
}

// DownloadAudio скачивает аудиофайл по URL во временную директорию.
// Возвращает путь к временному файлу (вызывающий обязан удалить его после использования).
func DownloadAudio(ctx context.Context, audioURL string, maxSize int64, logger *zap.Logger) (string, error) {
	logger.Info("download: fetching audio", zap.String("url", audioURL))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, audioURL, nil)
	if err != nil {
		return "", fmt.Errorf("download: new request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download: status %d for %s", resp.StatusCode, audioURL)
	}

	ext := filepath.Ext(audioURL)
	if ext == "" {
		ext = ".mp4"
	}

	tmp, err := os.CreateTemp("", "ai-audio-*"+ext)
	if err != nil {
		return "", fmt.Errorf("download: create temp: %w", err)
	}
	defer tmp.Close()

	limited := io.LimitReader(resp.Body, maxSize)
	written, err := io.Copy(tmp, limited)
	if err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("download: copy: %w", err)
	}

	logger.Info("download: saved audio", zap.String("path", tmp.Name()), zap.Int64("bytes", written))
	return tmp.Name(), nil
}

// whisperErrorResponse — структура ошибки от OpenAI API.
type whisperErrorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// parseWhisperError пытается распарсить JSON-ошибку от OpenAI.
func parseWhisperError(body []byte) string {
	var e whisperErrorResponse
	if json.Unmarshal(body, &e) == nil && e.Error.Message != "" {
		return e.Error.Message
	}
	return string(body)
}
