package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// SpeechServiceTranscriber — реализует TranscriptionProvider через speech-service.
// speech-service — self-hosted pipeline (faster-whisper + WhisperX + pyannote).
// Никаких платных API — всё локально.
type SpeechServiceTranscriber struct {
	baseURL string
	token   string
	client  *http.Client
	logger  *zap.Logger
}

func NewSpeechServiceTranscriber(baseURL, token string, logger *zap.Logger) *SpeechServiceTranscriber {
	return &SpeechServiceTranscriber{
		baseURL: baseURL,
		token:   token,
		client:  &http.Client{Timeout: 30 * time.Minute},
		logger:  logger,
	}
}

type speechProcessReq struct {
	EventID   string   `json:"event_id"`
	TenantID  string   `json:"tenant_id"`
	AudioURL  string   `json:"audio_url"`
	Language  string   `json:"language"`
	Diarize   bool     `json:"diarize"`
	ExportFmt []string `json:"export_formats"`
}

type speechJobResp struct {
	JobID   string `json:"job_id"`
	EventID string `json:"event_id"`
	Status  string `json:"status"`
}

type speechJobStatus struct {
	JobID  string `json:"job_id"`
	Status string `json:"status"` // queued | processing | done | failed
	Step   string `json:"step"`
	Error  string `json:"error"`
}

type speechResult struct {
	Transcript string `json:"transcript"`
	Language   string `json:"language_detected"`
}

// Transcribe делегирует транскрипцию в speech-service (self-hosted).
// Ждёт завершения задачи (polling) и возвращает текст транскрипции.
func (s *SpeechServiceTranscriber) Transcribe(ctx context.Context, audioPath, language string) (string, error) {
	s.logger.Info("speech-service: submitting job",
		zap.String("path", audioPath),
		zap.String("lang", language),
	)

	// Если путь локальный — speech-service должен иметь доступ к тому же тому
	audioURL := audioPath
	if len(audioPath) > 0 && audioPath[0] == '/' {
		audioURL = "file://" + audioPath
	}

	reqBody := speechProcessReq{
		EventID:   "internal",
		TenantID:  "internal",
		AudioURL:  audioURL,
		Language:  language,
		Diarize:   false,
		ExportFmt: []string{"json"},
	}

	body, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.baseURL+"/process", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("speech-service: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-Token", s.token)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("speech-service: submit: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("speech-service: submit status %d: %s", resp.StatusCode, string(b))
	}

	var jobResp speechJobResp
	if err = json.NewDecoder(resp.Body).Decode(&jobResp); err != nil {
		return "", fmt.Errorf("speech-service: decode job: %w", err)
	}

	return s.waitForResult(ctx, jobResp.JobID, jobResp.EventID)
}

func (s *SpeechServiceTranscriber) waitForResult(ctx context.Context, jobID, eventID string) (string, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			status, err := s.pollJob(ctx, jobID)
			if err != nil {
				s.logger.Warn("speech-service: poll error", zap.Error(err))
				continue
			}
			s.logger.Info("speech-service: job",
				zap.String("id", jobID),
				zap.String("status", status.Status),
				zap.String("step", status.Step),
			)
			switch status.Status {
			case "done":
				return s.fetchTranscript(ctx, eventID)
			case "failed":
				return "", fmt.Errorf("speech-service: job failed: %s", status.Error)
			}
		}
	}
}

func (s *SpeechServiceTranscriber) pollJob(ctx context.Context, jobID string) (*speechJobStatus, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/jobs/"+jobID, nil)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var status speechJobStatus
	if err = json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, err
	}
	return &status, nil
}

func (s *SpeechServiceTranscriber) fetchTranscript(ctx context.Context, eventID string) (string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/results/"+eventID, nil)
	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("speech-service: fetch result: %w", err)
	}
	defer resp.Body.Close()
	var result speechResult
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("speech-service: decode result: %w", err)
	}
	return result.Transcript, nil
}
