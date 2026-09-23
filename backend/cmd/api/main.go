package main

import (
	"context"
	"embed"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"akim-na-5-chasov/backend/internal/openai"
	"akim-na-5-chasov/backend/internal/simulation"
)

//go:embed web/index.html
var webFiles embed.FS

func jsonResponse(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
func main() {
	mux := http.NewServeMux()
	webRoot, err := fs.Sub(webFiles, "web")
	if err != nil {
		log.Fatal(err)
	}

	var aiClient *openai.Client
	if client, err := openai.NewClientFromEnv(); err == nil {
		aiClient = client
	} else {
		log.Printf("AI analysis disabled: %v", err)
	}

	mux.Handle("GET /", http.FileServer(http.FS(webRoot)))
	mux.HandleFunc("GET /api/catalog", func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, map[string]any{"budget": simulation.Budget, "districts": simulation.Districts, "initiatives": simulation.Initiatives, "directions": simulation.Directions()})
	})
	mux.HandleFunc("POST /api/simulate", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Choices []simulation.Choice `json:"choices"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonResponse(w, 400, map[string]string{"error": "некорректный JSON"})
			return
		}
		out, err := simulation.Simulate(body.Choices)
		if err != nil {
			jsonResponse(w, 422, map[string]string{"error": err.Error()})
			return
		}
		jsonResponse(w, 200, out)
	})
	mux.HandleFunc("POST /api/analyze", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Result simulation.Result `json:"result"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			jsonResponse(w, 400, map[string]string{"error": "некорректный JSON"})
			return
		}
		if aiClient == nil {
			jsonResponse(w, 200, map[string]any{"ok": false, "error": "AI analysis is temporarily unavailable."})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()
		analysis, err := aiClient.Analyze(ctx, body.Result)
		if err != nil {
			jsonResponse(w, 200, map[string]any{"ok": false, "error": "AI analysis is temporarily unavailable."})
			return
		}
		jsonResponse(w, 200, map[string]any{"ok": true, "analysis": analysis})
	})
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("API running at http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
