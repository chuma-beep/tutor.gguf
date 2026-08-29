# Gate-1 Media Kit — screenshots & demo video

Everything ADTC Gate 1 needs beyond the repo: the screenshot set and the 2-minute
demo video. Deadline: **Aug 25, 2026**.

---

## 1. Screenshot shot list (`docs/screenshots/`)

Capture at a clean terminal font size, light or dark theme consistent across shots.
Save as `docs/screenshots/NN-description.png`.

| # | Shot | Status | File |
|---|---|---|---|
| 01 | TUI streaming a calculus solution | ✅ captured | `docs/screenshots/01-tui-calculus.png` |
| 02 | Induction proof in the TUI | ✅ captured | `docs/screenshots/02-tui-induction.png` |
| 03 | JAMB-style word problem (`tp_001`) | ✅ captured | `docs/screenshots/03-tui-jamb-word-problem.png` |
| 04 | ASCII fallback mode | ✅ captured | `docs/screenshots/04-tui-ascii-fallback.png` |
| 05 | Raw API response (JSON `content` + `answer`) | ✅ captured | `docs/screenshots/05-api-json-response.png` |
| 06 | Retrieval transparency (chunks + prompt) | ✅ captured | `docs/screenshots/06-retrieval-transparency.png` |
| 07 | Desktop Wails window (Svelte + KaTeX) | ✅ captured | `docs/screenshots/07-desktop-wails-chat.png` |

Recapture instructions (how each was produced):

- 01: `bin/tutor chat -tutor-url http://localhost:8086` on a running `tutor serve`; ask `find the derivative of x^2`
- 02: same session, `Ctrl+L`, ask `Prove by induction that 1 + 2 + ... + n = n(n+1)/2`
- 03: same session, `Ctrl+L`, ask `tp_001` verbatim
- 04: `bin/tutor chat -ascii -tutor-url http://localhost:8086`, ask `find the derivative of x^2`
- 05: `curl -s http://localhost:8086/v1/complete -H 'Content-Type: application/json' -d '{"problem":"find x such that log_12(3x) = 2"}' | python3 -m json.tool`
- 06: `make run Q="Integrate 2x * e^(x^2)." EMBEDDER_URL=http://localhost:8085` (shows retrieved chunks + final ChatML prompt)
- 07: desktop binary built with `-tags "desktop production webkit2_41"`, window on Hyprland, `wtype` query `find the derivative of x^2`

Tip: `kooha` or OBS for clips; `flameshot`/`grim` for stills. A 10–15 s screen
recording of shot 01 streaming doubles as a video asset.

## 2. Two-minute demo video script

Pacing: ~300 words total. Record the terminal segments first, narrate over them.

### [0:00–0:20] The problem
> In Nigeria, most university students can't afford cloud AI — API fees in naira,
> unstable fibre, unreliable power. Yet the laptop they already own can run a
> language model. This is Tutor.gguf: a fully offline math tutor for discrete math,
> calculus, and linear algebra, built for the Africa Deep Tech Challenge Standard
> Laptop — 8 GB RAM, integrated graphics, zero internet.

### [0:20–0:55] Live demo *(screen recording)*
> Everything you see runs locally on CPU. I ask it to differentiate x squared —
> it streams a step-by-step solution with proper math rendering.
> *(type the induction prompt)* It handles proofs. And it speaks the student's
> context — here's a JAMB exam-style problem on arithmetic progressions.
> *(show boxed final answer)* Every answer ends in an unambiguous boxed result.

### [0:55–1:25] How it works *(architecture diagram or README scroll)*
> Under the hood: Qwen2.5-Math 1.5B quantized to GGUF Q4_K_M on llama.cpp — the
> required runtime. Around it, a retrieval-augmented pipeline: a local embedding
> model indexes worked examples from GSM8K, Hendrycks MATH, and Rosen's Discrete
> Mathematics; a keyword classifier picks domain-specific instructions per question.
> No cloud, no API keys — the whole stack is localhost.

### [1:25–1:45] The numbers *(benchmark table on screen)*
> Measured with the official ADTC profiler under the audit profile: peak RAM of
> 1.1 gigabytes — about 16 percent of the 7-gigabyte budget — roughly 14 tokens
> per second, and no thermal throttling.

### [1:45–2:00] Journey & close
> Building this meant tuning threads and context for a 4-core budget, building a
> LaTeX-to-terminal renderer so math reads properly, and running 30-case evals
> on-device. It's open source, it runs offline, and it runs on the hardware
> Africa already has. Thank you.

Recording notes:
- Terminal recorder: `kooha --file=...` or OBS; 1920×1080, font ≥ 14 pt.
- Voiceover: any mic is fine; re-record section by section rather than one take.
- Export MP4 (H.264), ≤ 200 MB, name `tutor-gguf-demo.mp4`.
