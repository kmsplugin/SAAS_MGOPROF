package service

import "context"

// TranscriptionProvider — интерфейс транскрипции аудио в текст.
// Реализация: SpeechServiceTranscriber (self-hosted, бесплатно).
type TranscriptionProvider interface {
	Transcribe(ctx context.Context, audioPath, language string) (string, error)
}
