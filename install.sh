#!/usr/bin/env bash
# tutor.gguf installer (macOS / Linux).
#
# Installs the single-binary tutor into ~/.local/bin (override with
# TUTOR_PREFIX) and runs `tutor setup`, which provisions the prebuilt
# llama.cpp server, both GGUF models, the corpus, and the vector index.
#
#   curl -fsSL https://raw.githubusercontent.com/chuma-beep/tutor.gguf/main/install.sh | bash
#
# Windows users: download tutor-windows-amd64.exe from Releases instead,
# then run `tutor setup` and `tutor chat`.
set -euo pipefail

REPO="${TUTOR_REPO:-chuma-beep/tutor.gguf}"
PREFIX="${TUTOR_PREFIX:-$HOME/.local/bin}"

say() { printf '\033[1m==>\033[0m %s\n' "$*"; }
die() { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

command -v curl >/dev/null 2>&1 || die "curl is required"

OS="$(uname -s)"
ARCH="$(uname -m)"
case "$OS" in
  Darwin) GOOS="darwin" ;;
  Linux) GOOS="linux" ;;
  *) die "unsupported OS: $OS (see README for Windows steps)" ;;
esac
case "$ARCH" in
  x86_64 | amd64) GOARCH="amd64" ;;
  arm64 | aarch64) GOARCH="arm64" ;;
  *) die "unsupported architecture: $ARCH" ;;
esac

ASSET="tutor-${GOOS}-${GOARCH}"
say "Fetching latest release of ${REPO} (${ASSET})"

TAG="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" |
  grep -o '"tag_name": *"[^"]*"' | head -1 | sed 's/.*"\(v[^"]*\)".*/\1/')"
[ -n "$TAG" ] || die "could not determine latest release tag"

URL="https://github.com/${REPO}/releases/download/${TAG}/${ASSET}"
mkdir -p "$PREFIX"
curl -fL --progress-bar -o "${PREFIX}/tutor.part" "$URL"
mv "${PREFIX}/tutor.part" "${PREFIX}/tutor"
chmod +x "${PREFIX}/tutor"
say "Installed ${PREFIX}/tutor (${TAG})"

say "Running setup — downloads ~1.2 GB (models + llama.cpp + corpus), one time only"
"${PREFIX}/tutor" setup

case ":$PATH:" in
  *":$PREFIX:"*) ;;
  *)
    say "${PREFIX} is not on your PATH — fix now and persist it:"
    printf '    export PATH="%s:$PATH"          # run this in your current shell\n' "$PREFIX"
    printf '    echo '"'"'export PATH="%s:$PATH"'"'"' >> ~/.bashrc   # persist\n' "$PREFIX"
    ;;
esac

say "Done! Start tutoring with:"
printf '    %s/tutor chat\n' "$PREFIX"
