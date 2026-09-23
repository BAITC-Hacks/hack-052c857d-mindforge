package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"akim-na-5-chasov/backend/internal/simulation"
)

type Analysis struct {
	Summary         string   `json:"summary"`
	Strengths       []string `json:"strengths"`
	Risks           []string `json:"risks"`
	Tradeoffs       []string `json:"tradeoffs"`
	Synergies       []string `json:"synergies"`
	Recommendations []string `json:"recommendations"`
}

type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string
}

type responseTextItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type responseOutput struct {
	Type    string             `json:"type"`
	Content []responseTextItem `json:"content"`
}

type responsesAPIResponse struct {
	Output []responseOutput `json:"output"`
}

func NewClientFromEnv() (*Client, error) {
	key := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if key == "" {
		return nil, errors.New("OPENAI_API_KEY is not configured")
	}
	model := strings.TrimSpace(os.Getenv("OPENAI_MODEL"))
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &Client{
		apiKey:     key,
		model:      model,
		httpClient: &http.Client{Timeout: 45 * time.Second},
		baseURL:    "https://api.openai.com/v1",
	}, nil
}

func (c *Client) Analyze(ctx context.Context, result simulation.Result) (Analysis, error) {
	payload := map[string]any{
		"model": c.model,
		"input": []map[string]any{
			{
				"role":    "system",
				"content": "Ты — аналитик синтетического симулятора управления городом «Аким на 5 часов». Числовой результат уже рассчитан детерминированной моделью. Твоя задача — ТОЛЬКО интерпретировать переданный результат. Используй только факты из данных. Не придумывай числа и не пересчитывай Astana Quality of Life Score, баллы районов, показатели, бюджет, эффекты мер или синергии. Напиши ответ СТРОГО НА РУССКОМ ЯЗЫКЕ. Кратко и предметно опиши: 1) общий итог; 2) сильные стороны; 3) оставшиеся риски; 4) важные компромиссы; 5) сработавшие синергии; 6) рекомендации. Не утверждай, что синтетический симулятор предсказывает реальные последствия для Астаны.",
			},
			{
				"role":    "user",
				"content": []map[string]any{{"type": "input_text", "text": compactSimulationPrompt(result)}},
			},
		},
		"text": map[string]any{
			"format": map[string]any{
				"type":   "json_schema",
				"name":   "city_analysis",
				"strict": true,
				"schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"summary":         map[string]any{"type": "string"},
						"strengths":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"risks":           map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"tradeoffs":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"synergies":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"recommendations": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					},
					"required":             []string{"summary", "strengths", "risks", "tradeoffs", "synergies", "recommendations"},
					"additionalProperties": false,
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return Analysis{}, fmt.Errorf("build OpenAI request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/responses", bytes.NewReader(body))
	if err != nil {
		return Analysis{}, fmt.Errorf("prepare OpenAI request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return Analysis{}, fmt.Errorf("OpenAI request failed: %w", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Analysis{}, fmt.Errorf("read OpenAI response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(string(data))
		if len(message) > 600 {
			message = message[:600] + "…"
		}
		return Analysis{}, fmt.Errorf("OpenAI API error (%s): %s", resp.Status, message)
	}

	var apiResp responsesAPIResponse
	if err := json.Unmarshal(data, &apiResp); err != nil {
		return Analysis{}, fmt.Errorf("parse OpenAI response: %w", err)
	}

	text := extractOutputText(apiResp)
	if text == "" {
		return Analysis{}, errors.New("OpenAI returned empty output")
	}

	var analysis Analysis
	if err := json.Unmarshal([]byte(text), &analysis); err != nil {
		return Analysis{}, fmt.Errorf("malformed AI response: %w", err)
	}
	if strings.TrimSpace(analysis.Summary) == "" {
		return Analysis{}, errors.New("OpenAI response missing summary")
	}
	return analysis, nil
}

func compactSimulationPrompt(result simulation.Result) string {
	payload := map[string]any{
		"baseline_score": result.BaselineScore,
		"final_score":    result.Score,
		"score_delta":    result.Delta,
		"budget": map[string]any{
			"total":     result.Budget,
			"spent":     result.Spent,
			"remaining": result.Remaining,
		},
		"selected":                result.Selected,
		"districts":               result.Districts,
		"synergies":               result.Synergies,
		"critical_count":          result.CriticalCount,
		"baseline_critical_count": result.BaselineCritical,
	}

	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "simulation result unavailable"
	}
	return string(body)
}

func extractOutputText(resp responsesAPIResponse) string {
	var pieces []string
	for _, out := range resp.Output {
		for _, content := range out.Content {
			if content.Type == "output_text" && strings.TrimSpace(content.Text) != "" {
				pieces = append(pieces, content.Text)
			}
		}
	}
	return strings.Join(pieces, "\n")
}
