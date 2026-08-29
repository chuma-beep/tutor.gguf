# Paths — adjust if yours differ
LLAMA_BIN     := $(HOME)/Projects/llama.cpp/build/bin/llama-server
LLAMA_BENCH   := $(HOME)/Projects/llama.cpp/build/bin/llama-bench
GEN_MODEL     := $(HOME)/Projects/models/qwen2.5-math-1.5b-instruct-q4_k_m.gguf
JUDGE_MODEL   := $(HOME)/Projects/models/qwen2.5-3b-instruct-q4_k_m.gguf
EMBED_MODEL   := $(HOME)/Projects/models/nomic-embed-text-v1.5.Q4_K_M.gguf
GEN_PORT      := 8080
EMBED_PORT    := 8081
JUDGE_PORT    := 8083
EMBEDDER_URL  := http://localhost:$(EMBED_PORT)
HENDRYCKS_DIR := data/raw/hendrycks_math
GSM8K_FILE    := data/raw/gsm8k/train.jsonl
ROSEN_DIR     := data/raw/rosen
EVAL_DIR      := evals
SERVE_PORT    := 8082
Q             ?= "find the derivative of x^2"
THREADS       ?= 0
CTX           ?= 2048

# Single static binary built from ./cmd/tutor (subcommands: serve|index|chat)
BIN           := bin/tutor

.PHONY: build setup serve-gen serve-embed serve-judge serve-tutor index run tui tui-ascii eval eval-fresh eval-quality eval-view eval-sample profile profile-audit build-desktop build-desktop-wails release-desktop bundle-offline dev-desktop dev-desktop-nobind

# Build the single tutor binary (trimpath + stripped for release-style size)
build:
	go build -trimpath -ldflags "-s -w" -o $(BIN) ./cmd/tutor

# One-shot local provisioning via the binary itself (models + llama.cpp +
# corpus into ~/.tutor, then index). Idempotent.
setup:
	go run ./cmd/tutor setup

# Start the generation model (Qwen2.5-Math)
# THREADS=0 -> auto; use THREADS=4 in 4-core / Docker-audit environments (-t 4
# beat auto-8 under a 4-CPU quota: 16.6 vs 14.8 t/s, see docs/tuning.md).
# CTX=2048 is the tuned minimum: worst-case RAG prompt 1122 + 512 max output
# = 1634 tokens; 4096 adds ~60 MB KV cache for no accuracy gain (see docs/tuning.md).
serve-gen:
	$(LLAMA_BIN) -m $(GEN_MODEL) --port $(GEN_PORT) -t $(THREADS) -c $(CTX)

# Start the embedding model (nomic-embed-text)
# batch 8192 = nomic native context; 2048 rejected >2048-token docs (500).
serve-embed:
	$(LLAMA_BIN) -m $(EMBED_MODEL) --embeddings --batch-size 8192 --ubatch-size 8192 -c 8192 --port $(EMBED_PORT) -t $(THREADS)

# Start the judge model (Qwen2.5-3B-Instruct) for llm-rubric evals
serve-judge:
	$(LLAMA_BIN) -m $(JUDGE_MODEL) --port $(JUDGE_PORT)

# Start the RAG tutor server (wraps retrieval + generation)
serve-tutor:
	go run ./cmd/tutor serve \
		-embedder-url $(EMBEDDER_URL) \
		-gen-url http://localhost:$(GEN_PORT) \
		-port $(SERVE_PORT)

# Index the corpus into chromem-go
index:
	go run ./cmd/tutor index \
		-embedder-url $(EMBEDDER_URL) \
		-hendrycks-dir $(HENDRYCKS_DIR) \
		-gsm8k-file $(GSM8K_FILE) \
		-rosen-dir $(ROSEN_DIR) \
		-query "$(Q)"

# Run a test query against the tutor
run:
	go run ./cmd/tutor index -embedder-url $(EMBEDDER_URL) -query "$(Q)"

# Interactive Bubble Tea shell against a running serve-tutor (step 4)
tui:
	go run ./cmd/tutor chat -tutor-url http://localhost:$(SERVE_PORT)

# The same shell, with ASCII math fallbacks instead of Unicode symbols
tui-ascii:
	go run ./cmd/tutor chat -ascii -tutor-url http://localhost:$(SERVE_PORT)

# Generate a fresh sample of test cases from the corpus
eval-sample:
	cd $(EVAL_DIR) && python3 sample_tests.py && python3 convert_tests.py

# Run promptfoo eval against the running generation server
eval:
	cd $(EVAL_DIR) && promptfoo eval --output results_$(shell date +%Y%m%d_%H%M%S).json

# Run promptfoo eval without disk cache
eval-fresh:
	cd $(EVAL_DIR) && promptfoo eval --no-cache --output results_$(shell date +%Y%m%d_%H%M%S).json

# Run the qualitative + African use-case eval set (requires serve-judge)
eval-quality:
	cd $(EVAL_DIR) && promptfoo eval -c quality.yaml --no-cache --output results_quality_$(shell date +%Y%m%d_%H%M%S).json

# Open the promptfoo results dashboard
eval-view:
	cd $(EVAL_DIR) && promptfoo view

# Run the ADTC profiler locally (participant mode, skip accuracy)
profile:
	adtc-profiler run \
		--submission . \
		--mode participant \
		--output submission.json \
		--skip-accuracy

# Build the Wails desktop. On systems with webkit2gtk-4.1 (Arch, Ubuntu 24.04+)
# use the webkit2_41 tag so CGO finds pkg-config; wails build alone fails there
# (4.0 missing) and leaves a stale ar archive. On Ubuntu 22.04 (webkit2gtk-4.0)
# use build-desktop-wails instead.
build-desktop:
	npm --prefix frontend ci --legacy-peer-deps
	npm --prefix frontend run build
	CGO_ENABLED=1 go build -tags "desktop production webkit2_41" -trimpath -ldflags "-s -w" -o build/bin/tutor-desktop ./cmd/desktop

# Ubuntu 22.04 / webkit2gtk-4.0 path (wails build handles deps + bindings)
build-desktop-wails:
	npm --prefix frontend ci --legacy-peer-deps
	npm --prefix frontend run build
	wails build -clean -tags desktop -ldflags "-s -w"

# Production installers for all platforms (thin: ~15 MB, no models bundled).
# First launch auto-downloads ~1.2 GB, then runs 100% offline. Unsigned.
release-desktop:
	npm --prefix frontend ci --legacy-peer-deps
	npm --prefix frontend run build
	wails build -clean -tags desktop -ldflags "-s -w" -platform linux/amd64 -nsis || true
	wails build -clean -tags desktop -ldflags "-s -w" -platform windows/amd64 -nsis || true
	wails build -clean -tags desktop -ldflags "-s -w" -platform darwin/universal || true
	ls -lh build/bin/

# Fat offline pack for air-gapped labs: pre-seeded ~/.tutor (models + corpus +
# index), ~1.4 GB. Campus Wi-Fi once, then USB.
bundle-offline:
	go run ./cmd/tutor setup
	tar -C $$(go env HOME)/.tutor -czf build/bin/tutor-offline.tar.gz .

# Dev shell with hot reload (Svelte + Go)
dev-desktop:
	wails dev -tags desktop

# Dev without bindings generation (workaround for AppArmor noexec on /tmp)
# Use when 'wails dev' hits 'fork/exec .../wailsbindings: permission denied'
dev-desktop-nobind:
	wails dev -tags desktop -skipbindings

# Run the ADTC profiler with a real accuracy benchmark (Sacc estimate)
profile-audit:
	adtc-profiler run \
		--submission . \
		--mode audit \
		--accuracy-task gsm8k \
		--accuracy-limit 50 \
		--output audit.json
