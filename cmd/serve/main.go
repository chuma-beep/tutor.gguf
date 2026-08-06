package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"github.com/chuma-beep/tutor.gguf/internal/llm"
	"github.com/chuma-beep/tutor.gguf/internal/rag"
	chromem "github.com/philippgille/chromem-go"
)

type requestBody struct {
	Problem     string  `json:"problem"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float64 `json:"temperature"`
}

type responseBody struct {
	Content string `json:"content"`
}

func main() {
	embedderURL := flag.String("embedder-url", "http://localhost:8081", "embedding server")
	genURL := flag.String("gen-url", "http://localhost:8080", "generation server (Qwen)")
	dbPath := flag.String("db-path", "./data/chromem", "chromem-go DB path")
	port := flag.String("port", "8082", "HTTP port to listen on")
	flag.Parse()

	ctx := context.Background()

	embedder := rag.NewEmbedder(*embedderURL)
	db, err := chromem.NewPersistentDB(*dbPath, false)
	if err != nil {
		log.Fatalf("open chromem db: %v", err)
	}
	collection, err := db.GetOrCreateCollection("tutor-corpus", nil, embedder.EmbedDocument)
	if err != nil {
		log.Fatalf("get or create collection: %v", err)
	}

	classifier := rag.NewSubdomainClassifier()
	retriever := rag.NewRetriever(collection, classifier, embedder)
	genClient := llm.NewClient(*genURL)

	http.HandleFunc("/v1/complete", func(w http.ResponseWriter, r *http.Request) {
		var req requestBody
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"bad request"}`, 400)
			return
		}
		if req.Problem == "" {
			http.Error(w, `{"error":"missing problem"}`, 400)
			return
		}

		chunks, subdomain, err := retriever.Retrieve(ctx, req.Problem)
		if err != nil {
			log.Printf("ERROR retrieval failed: %v", err)
			chunks, subdomain = nil, "other"
		}

		fullPrompt := rag.BuildPrompt(req.Problem, chunks, subdomain)
		content, err := genClient.Complete(fullPrompt)
		if err != nil {
			http.Error(w, `{"error":"generation failed"}`, 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(responseBody{Content: content})
	})

	log.Printf("tutor RAG server listening on :%s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
