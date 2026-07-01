#!/bin/bash
set -e

MODEL_DIR="model"
MODEL_FILE="qwen2.5-math-1.5b-instruct-q4_k_m.gguf"
MODEL_PATH="$MODEL_DIR/$MODEL_FILE"
MODEL_URL="https://huggingface.co/bartowski/Qwen2.5-Math-1.5B-Instruct-GGUF/resolve/main/Qwen2.5-Math-1.5B-Instruct-Q4_K_M.gguf"

mkdir -p "$MODEL_DIR"

if [ -f "$MODEL_PATH" ]; then
  echo "Model already exists at $MODEL_PATH — skipping download."
  exit 0
fi

echo "Downloading model to $MODEL_PATH..."
wget "$MODEL_URL" -O "$MODEL_PATH"
echo "Done."
