---
title: HTTP API
order: 5
---

# HTTP API

`tutor serve` exposes a small JSON API (default `http://localhost:8082`). CORS is open on `/v1/*` for local development.

## GET /health and /v1/health

Liveness probe. Returns `{"status":"ok"}`.

## GET /v1/status

Readiness plus wiring. Example:

```bash
curl -s localhost:8082/v1/status
```

```json
{"ready":true,"dbChunks":7474,"genURL":"http://127.0.0.1:33379","embedURL":"http://127.0.0.1:46147"}
```

`genURL`/`embedURL` are the active llama-servers (random free ports in managed mode). The browser UI polls this for its record head.

## POST /v1/complete

Blocking retrieval plus generation. `max_tokens` and `temperature` are optional (defaults 512 / 0.1):

```bash
curl -s localhost:8082/v1/complete \
  -H 'Content-Type: application/json' \
  -d '{"problem":"find the derivative of x^2"}'
```

Response:

```json
{"content":"<step-by-step solution>","answer":"2x","answer_rule":"boxed"}
```

`answer` is a lightweight parse of the final answer; `answer_rule` names the rule that matched (`boxed`, `final`, `gsm8k`, `none`). When nothing matches, `answer` falls back to the full content with rule `none` — the web UI hides its Final badge in that case. Eval scoring uses only `content`.

## POST /v1/complete/stream

Server-sent events. First event carries retrieval metadata (`subdomain`, `category`, `prompt`, `chunks`) so the UI can render pills and citations immediately; then `content` deltas; then a terminal event with `content`, `answer`, `answer_rule` and `done`, followed by `[DONE]`. Errors arrive as `{"error": ...}` events.
