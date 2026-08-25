// Package runtime locates and supervises the local llama.cpp processes and
// owns the on-disk layout of tutor's downloaded artifacts (~/.tutor by
// default): GGUF models, the llama-server binary, the corpus snapshot, the
// persistent vector store, and server logs.
package runtime

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	// GenModelFile is the scored submission model (Qwen2.5-Math-1.5B-Instruct).
	GenModelFile = "qwen2.5-math-1.5b-instruct-q4_k_m.gguf"
	// EmbedModelFile is the retrieval embedding model (nomic-embed-text-v1.5).
	EmbedModelFile = "nomic-embed-text-v1.5.Q4_K_M.gguf"
)

// TutorHome returns the root directory for tutor's downloaded artifacts.
// Precedence: $TUTOR_HOME > ~/.tutor.
func TutorHome() string {
	if v := os.Getenv("TUTOR_HOME"); v != "" {
		return v
	}
	base, err := os.UserHomeDir()
	if err != nil {
		return ".tutor"
	}
	return filepath.Join(base, ".tutor")
}

func ModelsDir() string { return filepath.Join(TutorHome(), "models") }
func BinDir() string    { return filepath.Join(TutorHome(), "bin") }
func LogsDir() string   { return filepath.Join(TutorHome(), "logs") }
func CorpusDir() string { return filepath.Join(TutorHome(), "corpus") }

// DBPath is the default chromem-go store used by managed runs.
func DBPath() string { return filepath.Join(TutorHome(), "chromem") }

func GenModelPath() string   { return filepath.Join(ModelsDir(), GenModelFile) }
func EmbedModelPath() string { return filepath.Join(ModelsDir(), EmbedModelFile) }

// Corpus paths populated by `tutor setup`.
func GSM8KTrainFile() string    { return filepath.Join(CorpusDir(), "gsm8k", "train.jsonl") }
func HendrycksTrainDir() string { return filepath.Join(CorpusDir(), "hendrycks_math", "train") }
func RosenDir() string          { return filepath.Join(CorpusDir(), "rosen") }

// LlamaServerPath is where setup installs the prebuilt llama-server binary.
func LlamaServerPath() string {
	name := "llama-server"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(BinDir(), name)
}
