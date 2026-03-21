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

const (
	claudeAPIURL      = "https://api.anthropic.com/v1/messages"
	claudeAPIVersion  = "2023-06-01"
	claudeMaxTokens   = 4096
)

// ClaudeClient — HTTP-клиент к Anthropic API.
type ClaudeClient struct {
	apiKey string
	model  string
	client *http.Client
	logger *zap.Logger
}

func NewClaudeClient(apiKey, model string, logger *zap.Logger) *ClaudeClient {
	return &ClaudeClient{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{Timeout: 5 * time.Minute},
		logger: logger,
	}
}

type claudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system,omitempty"`
	Messages  []claudeMessage `json:"messages"`
}

type claudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type claudeResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// Complete отправляет запрос к Claude и возвращает текстовый ответ.
func (c *ClaudeClient) Complete(ctx context.Context, system, prompt string) (string, error) {
	payload := claudeRequest{
		Model:     c.model,
		MaxTokens: claudeMaxTokens,
		System:    system,
		Messages: []claudeMessage{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("claude: marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, claudeAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("claude: new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", claudeAPIVersion)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("claude: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("claude: read body: %w", err)
	}

	var cr claudeResponse
	if err = json.Unmarshal(respBody, &cr); err != nil {
		return "", fmt.Errorf("claude: unmarshal response: %w", err)
	}

	if cr.Error != nil {
		return "", fmt.Errorf("claude: api error [%s]: %s", cr.Error.Type, cr.Error.Message)
	}

	if len(cr.Content) == 0 {
		return "", fmt.Errorf("claude: empty response")
	}

	text := cr.Content[0].Text
	c.logger.Info("claude: got response", zap.Int("chars", len(text)))
	return text, nil
}

// ────────────────────────────────────────────────────────────
// Промпты для каждого типа AI-контента
// ────────────────────────────────────────────────────────────

const systemPrompt = `Ты — эксперт-аналитик встреч и вебинаров.
Ты получаешь транскрипцию записи и создаёшь структурированный,
профессиональный анализ на том же языке, что и транскрипция.
Отвечай строго по теме, без вводных фраз вроде "Вот анализ:" или "Конечно!".`

// SummaryPrompt возвращает промпт для краткого резюме.
func SummaryPrompt(transcript string) string {
	return fmt.Sprintf(`Создай краткое и информативное резюме следующей встречи/вебинара (3-5 абзацев).
Включи: главную тему, ключевые обсуждения, принятые решения, итоги.

ТРАНСКРИПЦИЯ:
%s`, transcript)
}

// HighlightsPrompt возвращает промпт для ключевых моментов.
func HighlightsPrompt(transcript string) string {
	return fmt.Sprintf(`Извлеки 5-10 ключевых моментов и инсайтов из транскрипции встречи/вебинара.
Формат: нумерованный список. Каждый пункт — одно-два предложения.
Включи только действительно важные моменты, цитаты, числа, факты.

ТРАНСКРИПЦИЯ:
%s`, transcript)
}

// ChaptersPrompt возвращает промпт для разбивки на главы.
func ChaptersPrompt(transcript string) string {
	return fmt.Sprintf(`Разбей транскрипцию встречи на логические главы/разделы с временными метками.
Поскольку точных временных меток нет — оцени примерно по позиции в тексте.

Формат каждой главы:
## [Название главы]
**Примерная позиция:** начало / середина / конец (или %, например "~25%%")
**Содержание:** 2-3 предложения о чём шла речь в этом разделе.

ТРАНСКРИПЦИЯ:
%s`, transcript)
}

// ActionItemsPrompt возвращает промпт для списка задач.
func ActionItemsPrompt(transcript string) string {
	return fmt.Sprintf(`Извлеки все задачи, договорённости, дедлайны и поручения из транскрипции встречи.

Формат каждой задачи:
- [ ] **Задача:** описание
  **Ответственный:** имя (если упоминается) или "не указан"
  **Срок:** дата/срок (если упоминается) или "не указан"

Если задач нет — напиши "Явных задач в записи не выявлено."

ТРАНСКРИПЦИЯ:
%s`, transcript)
}
