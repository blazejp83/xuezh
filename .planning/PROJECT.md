# xuezh — Local Chinese Learning Engine

## What This Is

A CLI-based Mandarin learning engine that runs as a skill for Clawdbot on Mac Mini M4. It handles spaced repetition scheduling, vocabulary tracking, pronunciation assessment, and audio synthesis — while the LLM agent provides all pedagogy and teaching decisions. Forked from joshp123/xuezh to add local TTS support.

## Core Value

Reliable, low-latency Chinese TTS that works locally on Apple Silicon without internet dependency, controllable via CLI by an AI agent.

## Requirements

### Validated

- SRS scheduling (SM-2/Leitner) with dual recall/pronunciation tracks — existing
- HSK vocabulary/character/grammar import and tracking — existing
- JSON envelope CLI contract for agent consumption — existing
- Audio format conversion via ffmpeg — existing
- Pronunciation assessment via Azure Speech — existing
- Pluggable audio backend architecture — existing

### Active

- [ ] Local TTS backend using Qwen3-TTS via mlx-audio on Apple Silicon
- [ ] TTS server lifecycle management (`audio server start/stop/status`)
- [ ] Agent can start TTS server, get notified when model is warm, use it, then stop it
- [ ] Keep edge-tts as fallback backend (internet-dependent)
- [ ] Server health check and PID tracking in workspace

### Out of Scope

- Multi-user support — single learner, single machine
- Web UI or API server for the engine itself — CLI only, consumed by agent
- Custom voice training or voice cloning — use stock voices
- Replacing Azure Speech for pronunciation assessment — separate from TTS
- Running on non-Apple Silicon hardware — M4 Mac Mini is the target

## Context

- **Target hardware:** Mac Mini M4 (Apple Silicon, unified memory, Metal/MPS)
- **TTS model:** Qwen3-TTS 1.7B via mlx-audio (MLX framework, ~2-3 GB RAM)
- **Integration:** mlx-audio provides OpenAI-compatible HTTP server at localhost
- **Cold start:** Model loading takes 10-30s, so server must persist between TTS calls
- **Agent flow:** Agent starts server at session begin, makes TTS calls during lesson, stops at end
- **Existing codebase:** ~4,600 LOC Go, well-structured with pluggable backends
- **Only ~920 LOC (audio module) needs modification** — rest of codebase untouched

## Constraints

- **Platform:** macOS ARM64 (Apple Silicon M4) — mlx-audio requires this
- **Architecture:** Two processes required — Go CLI (stateless) + mlx-audio server (persistent, holds model)
- **Communication:** HTTP to localhost (mlx-audio exposes OpenAI-compatible `/v1/audio/speech`)
- **Technology:** Go for CLI, Python/MLX for TTS server (managed as subprocess)
- **Backwards compatibility:** Existing CLI contract must not break — new commands additive only

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Keep Go codebase, don't rewrite | 90% of code unrelated to TTS; rewrite wastes 2-3 weeks | — Pending |
| Qwen3-TTS via mlx-audio | Best Chinese quality + native Apple Silicon optimization | — Pending |
| HTTP localhost for TTS communication | mlx-audio already provides OpenAI-compatible server | — Pending |
| Server lifecycle in CLI | Agent needs single interface; no launchd/systemd knowledge needed | — Pending |
| Keep edge-tts as fallback | Zero-cost fallback when internet available | — Pending |

---
*Last updated: 2026-02-28 after initialization*
