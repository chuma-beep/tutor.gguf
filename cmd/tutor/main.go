package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/chuma-beep/tutor.gguf/internal/rag"
	chromem "github.com/philippgille/chromem-go"
)

func main() {
	var (
		embedderURL  = flag.String("embedder-url", "http://localhost:8080", "llama.cpp server running nomic-embed-text (--embedding mode)")
		dbPath       = flag.String("db-path", "./data/chromem", "path for the persistent chromem-go DB")
		hendrycksDir = flag.String("hendrycks-dir", "", "directory of Hendrycks MATH JSON files (optional)")
		gsm8kFile    = flag.String("gsm8k-file", "", "path to GSM8K JSONL file (optional)")
		rosenDir     = flag.String("rosen-dir", "", "directory of Rosen .md/.txt files (optional)")
		query        = flag.String("query", "", "test query to run after indexing")
	)
	flag.Parse()

	if *query == "" {
		log.Fatal("pass a test query with -query, e.g. -query \"find the derivative of x^2\"")
	}

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

	// --- Load + index whatever corpus sources were passed in ---
	var chunks []rag.Chunk

	if *hendrycksDir != "" {
		loaded, err := loadHendrycksDir(*hendrycksDir)
		if err != nil {
			log.Fatalf("load hendrycks dir: %v", err)
		}
		chunks = append(chunks, loaded...)
		fmt.Printf("loaded %d Hendrycks chunks\n", len(loaded))
	}

	if *gsm8kFile != "" {
		loaded, err := rag.LoadGSM8KFile(*gsm8kFile)
		if err != nil {
			log.Fatalf("load gsm8k file: %v", err)
		}
		chunks = append(chunks, loaded...)
		fmt.Printf("loaded %d GSM8K chunks\n", len(loaded))
	}

	if *rosenDir != "" {
		loaded, err := rag.LoadRosenDir(*rosenDir)
		if err != nil {
			log.Fatalf("load rosen dir: %v", err)
		}
		chunks = append(chunks, loaded...)
		fmt.Printf("loaded %d Rosen chunks\n", len(loaded))
	}

	if len(chunks) == 0 {
		fmt.Println("no corpus sources passed — assuming the collection was already indexed in a previous run, skipping ingestion")
	} else {
		docs := make([]chromem.Document, 0, len(chunks))
		for i, c := range chunks {
			docs = append(docs, chromem.Document{
				ID:      fmt.Sprintf("%s-%d", c.Source, i),
				Content: c.Text,
				Metadata: map[string]string{
					"subdomain": c.Subdomain,
					"source":    c.Source,
					"level":     c.Level,
				},
			})
		}

		fmt.Printf("embedding + indexing %d chunks (this calls the embedder once per chunk, sequentially)...\n", len(docs))
		if err := collection.AddDocuments(ctx, docs, 1); err != nil {
			log.Fatalf("add documents: %v", err)
		}
		fmt.Println("indexing done")
	}

	// --- Run a real query through the full retrieval + prompt flow ---
	classifier := rag.NewSubdomainClassifier()
	retriever := rag.NewRetriever(collection, classifier, embedder)

	results, err := retriever.Retrieve(ctx, *query)
	if err != nil {
		log.Fatalf("retrieve: %v", err)
	}

	fmt.Printf("\n--- retrieved %d chunks for query: %q ---\n", len(results), *query)
	for i, r := range results {
		fmt.Printf("[%d] subdomain=%s similarity=%.4f source=%s\n%s\n\n", i+1, r.Subdomain, r.Similarity, r.Source, truncate(r.Text, 200))
	}

	prompt := rag.BuildPrompt(*query, results)
	fmt.Println("--- final prompt sent to Qwen2.5-Math ---")
	fmt.Println(prompt)
}

// loadHendrycksDir applies LoadHendrycksFile across every .json file in a
// directory, since the chunker only handles one file at a time.
func loadHendrycksDir(dir string) ([]rag.Chunk, error) {
	entries, err := readJSONFiles(dir)
	if err != nil {
		return nil, err
	}
	var chunks []rag.Chunk
	for _, path := range entries {
		c, err := rag.LoadHendrycksFile(path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		chunks = append(chunks, c)
	}
	return chunks, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// readJSONFiles returns full paths to every .json file in dir (non-recursive).
func readJSONFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}
	var paths []string
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		paths = append(paths, filepath.Join(dir, e.Name()))
	}
	return paths, nil
}
