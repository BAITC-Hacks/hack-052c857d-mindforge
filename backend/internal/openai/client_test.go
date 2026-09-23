package openai

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"akim-na-5-chasov/backend/internal/simulation"
)

func TestNewClientFromEnv(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("OPENAI_MODEL", "gpt-4.1-mini")

	client, err := NewClientFromEnv()
	if err != nil {
		t.Fatalf("NewClientFromEnv returned error: %v", err)
	}
	if client.model != "gpt-4.1-mini" {
		t.Fatalf("model = %q, want gpt-4.1-mini", client.model)
	}
}

func TestNewClientFromEnvMissingKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	_, err := NewClientFromEnv()
	if err == nil {
		t.Fatal("expected missing key error")
	}
}

func TestClientAnalyzeValidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/responses" {
			t.Fatalf("wrong path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization header = %q, want Bearer test-key", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"output":[{"type":"message","content":[{"type":"output_text","text":"{\"summary\":\"Overall result is strong\",\"strengths\":[\"Good\"],\"risks\":[\"Risk\"],\"tradeoffs\":[\"Trade\"],\"synergies\":[\"M10+M12\"],\"recommendations\":[\"Keep investing\"]}"}]}]}`)
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-key",
		model:      "gpt-4.1-mini",
		httpClient: server.Client(),
		baseURL:    server.URL + "/v1",
	}

	analysis, err := client.Analyze(context.Background(), simulation.Result{
		Budget:        100,
		Spent:         95,
		Remaining:     5,
		BaselineScore: 52.56,
		Score:         56.5,
		Delta:         3.94,
		Selected:      []simulation.Choice{{InitiativeID: "M7", DistrictID: "nura"}},
	})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if analysis.Summary == "" || len(analysis.Strengths) == 0 {
		t.Fatal("analysis was not decoded correctly")
	}
}

func TestClientAnalyzeMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"output":[{"type":"message","content":[{"type":"output_text","text":"not-json"}]}]}`)
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-key",
		model:      "gpt-4.1-mini",
		httpClient: server.Client(),
		baseURL:    server.URL + "/v1",
	}

	_, err := client.Analyze(context.Background(), simulation.Result{Budget: 100})
	if err == nil || !strings.Contains(err.Error(), "malformed AI response") {
		t.Fatalf("expected malformed AI response error, got %v", err)
	}
}

func TestClientAnalyzeTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-key",
		model:      "gpt-4.1-mini",
		httpClient: &http.Client{Timeout: 10 * time.Millisecond},
		baseURL:    server.URL + "/v1",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := client.Analyze(ctx, simulation.Result{Budget: 100})
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestClientAnalyzeHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	client := &Client{
		apiKey:     "test-key",
		model:      "gpt-4.1-mini",
		httpClient: server.Client(),
		baseURL:    server.URL + "/v1",
	}

	_, err := client.Analyze(context.Background(), simulation.Result{Budget: 100})
	if err == nil || !strings.Contains(err.Error(), "OpenAI API error") {
		t.Fatalf("expected OpenAI API error, got %v", err)
	}
}
