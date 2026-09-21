---
title: Getting started
order: 1
---

# Getting started

Tutor.gguf is a fully offline mathematics tutor for Nigerian computer science undergraduates. Give it a math problem and it writes a step-by-step solution — on the laptop you already own. No GPU, no internet, no cloud account, no per-token fee.

## What you need

The reference machine is an 8 GB laptop with integrated graphics (Ubuntu 22.04 is the reference OS). The tutor peaks around 1.1 GB RAM, well under the 7 GB budget. Desktop builds also exist for macOS and Windows.

## Install paths

Pick one:

**Prebuilt CLI (Linux/macOS).** Fetch the latest release binary and provision everything:

```bash
curl -fsSL https://raw.githubusercontent.com/chuma-beep/tutor.gguf/main/install.sh | bash
```

This installs `tutor` into `~/.local/bin` and runs `tutor setup` (about 1.2 GB of models, llama.cpp and corpus — one time only).

**Windows.** Download `tutor-windows-amd64.exe` from Releases, then run `tutor setup` and `tutor chat`.

**Desktop app.** Download the `.deb`, `.AppImage`, `.dmg` or Windows installer from Releases. Double-click to install, then just chat: first launch downloads the models with a progress bar and resumes after interruptions. After that it runs 100% offline forever.

**From source.** Requires Go 1.26+, Node 20+ (desktop UI only) and Python 3.11+ with promptfoo (evals only):

```bash
git clone https://github.com/chuma-beep/tutor.gguf
cd tutor.gguf
./download_model.sh   # Qwen2.5-Math into model/
```

## Next

Run [Setup](setup) once, then open [Using the tutor](using).
