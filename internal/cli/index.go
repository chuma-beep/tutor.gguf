package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/chuma-beep/tutor.gguf/internal/rag"
	chromem "github.com/philippgille/chromem-go"
)

// IndexOptions parameterizes corpus ingestion. RunQuery controls whether a
// test query is retrieved and its final prompt printed after indexing.
type IndexOptions struct {
	EmbedderURL  string
	DBPath       string
	HendrycksDir string // optional
	GSM8KFile    string // optional
	RosenDir     string // optional
	Query        string
	RunQuery     bool
}

// Index ingests corpus sources into the persistent chromem-go store, then runs
// a test query through retrieval + prompt building so the caller can inspect
// exactly what would be sent to the generation model.
func Index(args []string) error {
	fs := flag.NewFlagSet("index", flag.ContinueOnError)
	var (
		embedderURL  = fs.String("embedder-url", "http://localhost:8080", "llama.cpp server running nomic-embed-text (--embedding mode)")
		dbPath       = fs.String("db-path", "", "path for the persistent chromem-go DB")
		hendrycksDir = fs.String("hendrycks-dir", "", "directory of Hendrycks MATH JSON files (optional)")
		gsm8kFile    = fs.String("gsm8k-file", "", "path to GSM8K JSONL file (optional)")
		rosenDir     = fs.String("rosen-dir", "", "directory of Rosen .md/.txt files (optional)")
		query        = fs.String("query", "", "test query to run after indexing")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *query == "" {
		return errors.New(`pass a test query with -query, e.g. tutor index -query "find the derivative of x^2"`)
	}

	return RunIndex(context.Background(), IndexOptions{
		EmbedderURL:  *embedderURL,
		DBPath:       resolveDBPath(*dbPath),
		HendrycksDir: *hendrycksDir,
		GSM8KFile:    *gsm8kFile,
		RosenDir:     *rosenDir,
		Query:        *query,
		RunQuery:     true,
	})
}

// RunIndex is the ingestion core shared by `tutor index` and `tutor setup`.
func RunIndex(ctx context.Context, o IndexOptions) error {
	embedder := rag.NewEmbedder(o.EmbedderURL)

	db, err := chromem.NewPersistentDB(o.DBPath, false)
	if err != nil {
		return fmt.Errorf("open chromem db: %w", err)
	}

	collection, err := db.GetOrCreateCollection("tutor-corpus", nil, embedder.EmbedDocument)
	if err != nil {
		return fmt.Errorf("get or create collection: %w", err)
	}

	// --- Load + index whatever corpus sources were passed in ---
	var chunks []rag.Chunk

	if o.HendrycksDir != "" {
		loaded, err := loadHendrycksDir(o.HendrycksDir)
		if err != nil {
			return fmt.Errorf("load hendrycks dir: %w", err)
		}
		chunks = append(chunks, loaded...)
		fmt.Printf("loaded %d Hendrycks chunks\n", len(loaded))
	}

	if o.GSM8KFile != "" {
		loaded, err := rag.LoadGSM8KFile(o.GSM8KFile)
		if err != nil {
			return fmt.Errorf("load gsm8k file: %w", err)
		}
		chunks = append(chunks, loaded...)
		fmt.Printf("loaded %d GSM8K chunks\n", len(loaded))
	}

	if o.RosenDir != "" {
		loaded, err := rag.LoadRosenDir(o.RosenDir)
		if err != nil {
			return fmt.Errorf("load rosen dir: %w", err)
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
			return fmt.Errorf("add documents: %w", err)
		}
		fmt.Println("indexing done")
	}

	if !o.RunQuery {
		return nil
	}

	// --- Run a real query through the full retrieval + prompt flow ---
	classifier := rag.NewSubdomainClassifier()
	retriever := rag.NewRetriever(collection, classifier, embedder)

	results, subdomain, err := retriever.Retrieve(ctx, o.Query)
	if err != nil {
		return fmt.Errorf("retrieve: %w", err)
	}

	fmt.Printf("\n--- retrieved %d chunks for query: %q ---\n", len(results), o.Query)
	for i, r := range results {
		fmt.Printf("[%d] subdomain=%s similarity=%.4f source=%s\n%s\n\n", i+1, r.Subdomain, r.Similarity, r.Source, truncate(r.Text, 200))
	}

	prompt := rag.BuildPrompt(o.Query, results, subdomain)
	fmt.Println("--- final prompt sent to Qwen2.5-Math ---")
	fmt.Println(prompt)
	return nil
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
