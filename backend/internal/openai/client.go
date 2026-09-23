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

type responsesRequest struct {
	Model          string         `json:"model"`
	Input          any            `json:"input"`
	Temperature    float64        `json:"temperature,omitempty"`
	ResponseFormat map[string]any `json:"response_format,omitempty"`
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
		model = "gpt-4.1-mini"
	}
	return &Client{
		apiKey:     key,
		model:      model,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    "https://api.openai.com/v1",
	}, nil
}

func (c *Client) Analyze(ctx context.Context, result simulation.Result) (Analysis, error) {
	payload := map[string]any{
		"model": c.model,
		"input": []map[string]any{
			{
				"role":    "system",
				"content": "You are an analytical advisor for a synthetic city-management simulator called 'Аким на 5 часов'. The numerical simulation has already been calculated by a deterministic engine. Your job is ONLY to interpret the supplied simulation result. Use only facts present in the supplied data. Never invent numbers. Never recalculate or replace Astana Quality of Life Score, district scores, indicator values, budget, initiative effects, or synergy effects. Analyze the strategy and explain: 1. overall result; 2. strengths; 3. remaining risks; 4. important trade-offs; 5. activated synergies; 6. possible alternative strategic directions. Do not claim that this synthetic simulator predicts actual real-world outcomes for Astana. Keep the analysis concise, specific, and grounded in the supplied simulation data.",
			},
			{
				"role":    "user",
				"content": []map[string]any{{"type": "input_text", "text": compactSimulationPrompt(result)}},
			},
		},
		"temperature": 0.2,
		"response_format": map[string]any{
			"type": "json_schema",
			"json_schema": map[string]any{
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
		return Analysis{}, fmt.Errorf("OpenAI API error: status %d", resp.StatusCode)
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
