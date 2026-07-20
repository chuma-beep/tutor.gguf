# Paths — adjust if yours differ
LLAMA_BIN     := $(HOME)/Projects/llama.cpp/build/bin/llama-server
GEN_MODEL     := $(HOME)/Projects/models/qwen2.5-math-1.5b-instruct-q4_k_m.gguf
EMBED_MODEL   := $(HOME)/Projects/models/nomic-embed-text-v1.5.Q4_K_M.gguf

GEN_PORT      := 8080
EMBED_PORT    := 8081
EMBEDDER_URL  := http://localhost:$(EMBED_PORT)

HENDRYCKS_DIR := data/raw/hendrycks_math
GSM8K_FILE    := data/raw/gsm8k/train.jsonl
ROSEN_DIR     := data/raw/rosen

.PHONY: serve-gen serve-embed index run

# Start the generation model (Qwen2.5-Math)
serve-gen:
	$(LLAMA_BIN) -m $(GEN_MODEL) --port $(GEN_PORT)

# Start the embedding model (nomic-embed-text)
serve-embed:
	$(LLAMA_BIN) -m $(EMBED_MODEL) --embeddings --port $(EMBED_PORT)

# Index the corpus into chromem-go
index:
	go run ./cmd/index \
		-embedder-url $(EMBEDDER_URL) \
		-hendrycks-dir $(HENDRYCKS_DIR) \
		-gsm8k-file $(GSM8K_FILE) \
		-rosen-dir $(ROSEN_DIR) \
		-query "$(Q)"

# Run a test query against the tutor
run:
	go run ./cmd/tutor -embedder-url $(EMBEDDER_URL) -query "$(Q)"
