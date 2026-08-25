package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/chuma-beep/tutor.gguf/internal/llm"
	"github.com/chuma-beep/tutor.gguf/internal/parse"
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
	Answer  string `json:"answer,omitempty"`
}

// NewRAGHandler builds the /v1/complete handler: retrieval + subdomain
// classification + prompt build + blocking generation call.
func NewRAGHandler(ctx context.Context, embedderURL, genURL, dbPath string) (http.Handler, error) {
	embedder := rag.NewEmbedder(embedderURL)
	db, err := chromem.NewPersistentDB(dbPath, false)
	if err != nil {
		return nil, fmt.Errorf("open chromem db: %w", err)
	}
	collection, err := db.GetOrCreateCollection("tutor-corpus", nil, embedder.EmbedDocument)
	if err != nil {
		return nil, fmt.Errorf("get or create collection: %w", err)
	}

	classifier := rag.NewSubdomainClassifier()
	retriever := rag.NewRetriever(collection, classifier, embedder)
	genClient := llm.NewClient(genURL)

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/complete", func(w http.ResponseWriter, r *http.Request) {
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
		json.NewEncoder(w).Encode(responseBody{Content: content, Answer: parse.Extract(content)})
	})
	return mux, nil
}

// Serve runs the RAG HTTP server. With no -gen-url/-embedder-url it starts and
// supervises local llama-server processes itself; with both URLs set it acts
// as a plain client against external servers (the dev Makefile flow).
func Serve(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	embedderURL := fs.String("embedder-url", "", "embedding server URL — omit BOTH urls to auto-start local llama-servers")
	genURL := fs.String("gen-url", "", "generation server URL (Qwen) — omit BOTH urls to auto-start local llama-servers")
	dbPath := fs.String("db-path", "", "chromem-go DB path (default: ./data/chromem in a repo checkout, else ~/.tutor/chromem)")
	port := fs.String("port", "8082", "HTTP port to listen on")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if (*embedderURL == "") != (*genURL == "") {
		return errors.New("pass -embedder-url AND -gen-url together for external servers, or neither to start them automatically")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var (
		handler http.Handler
		err     error
		mgr     *serverManager
	)
	if *genURL == "" {
		mgr, err = startManaged(ctx)
		if err != nil {
			return err
		}
		defer mgr.stop()
		handler, err = NewRAGHandler(ctx, mgr.embedURL, mgr.genURL, resolveDBPath(*dbPath))
	} else {
		handler, err = NewRAGHandler(ctx, *embedderURL, *genURL, resolveDBPath(*dbPath))
	}
	if err != nil {
		return err
	}

	addr := ":" + *port
	srv := &http.Server{Addr: addr, Handler: handler}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	log.Printf("tutor RAG server listening on %s", addr)
	select {
	case err := <-errCh:
		if mgr != nil {
			mgr.stop()
		}
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("listen on %s: %w", addr, err)
	case <-ctx.Done():
		log.Printf("shutting down")
		srv.Close()
		if mgr != nil {
			mgr.stop()
		}
		return nil
	}
}
