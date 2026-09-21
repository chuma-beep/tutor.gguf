---
title: Using the tutor
order: 3
---

# Using the tutor

Three frontends share one backend: the terminal shell, the browser UI and the desktop app.

## Terminal shell

```bash
tutor chat            # Unicode math: stacked fractions, tall brackets, boxed answers
tutor chat -ascii     # ASCII-only fallbacks for exotic fonts
```

Type a question and press Enter; the answer streams into the transcript. `PgUp`/`PgDn` scroll, `Ctrl+L` clears, `Esc` or `Ctrl+C` quits. This is the best option over SSH or on machines with no desktop environment.

## Browser UI

```bash
tutor serve           # serves the API plus the web UI at http://localhost:8082/
```

Open the URL: a record head shows live status (readiness, indexed examples, model), each answer lands as a numbered plate with its category pill, and the right rail shows the retrieved chunks behind every answer. Per-turn actions: copy answer/question, retry, delete. The input box takes multiple lines (`Shift+Enter`), streams with a Stop button, and caps at 2000 characters. History persists in the browser (100 turns, `Ctrl+L` clears). Toggle light/dark/system theme from the header.

## Desktop app

Same engine as the browser UI in a native window, no terminal and no commands. First launch provisions models with a progress bar; afterwards it is offline forever.

## Asking well

Write the full problem, including constraints ("using π = 22/7", "by induction", "using matrix row reduction"). The tutor classifies each question (calculus, discrete math, linear algebra, geometry, probability, number theory) to focus retrieval and pick prompt instructions — the pill on each answer shows what it chose.

## The Final badge

When the model states its answer in a recognized form (`\boxed{}`, `final answer:`, `####`), it appears in an amber badge under the solution. No badge means the model never committed to a final form — the reasoning above is still the answer.
