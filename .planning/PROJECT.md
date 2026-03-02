# xuezh — Local Chinese Learning Engine

## What This Is

A CLI-based Mandarin learning engine that runs as a skill for Clawdbot on Mac Mini M4. It handles spaced repetition scheduling, vocabulary tracking, pronunciation assessment, and audio synthesis — while the LLM agent provides all pedagogy and teaching decisions. Forked from joshp123/xuezh to add local TTS and STT support via mlx-audio on Apple Silicon.

## Core Value

Reliable, low-latency Chinese TTS and STT that works locally on Apple Silicon without internet dependency, controllable via CLI by an AI agent.

## Requirements

### Validated

- SRS scheduling (SM-2/Leitner) with dual recall/pronunciation tracks — existing
- HSK vocabulary/character/grammar import and tracking — existing
- JSON envelope CLI contract for agent consumption — existing
- Audio format conversion via ffmpeg — existing
- Pronunciation assessment via Azure Speech — existing
- Pluggable audio backend architecture — existing
- Local TTS backend using Qwen3-TTS via mlx-audio on Apple Silicon — v1.0
- TTS server lifecycle management (`audio server start/stop/status`) — v1.0
- Agent can start TTS server, get notified when model is warm, use it, then stop it — v1.0
- Keep edge-tts as fallback backend (internet-dependent) — v1.0
- Server health check and PID tracking in workspace — v1.0
- Local STT via HTTP POST to mlx-audio /v1/audio/transcriptions — v1.1
- STT response parsed into structured transcript with segments and timestamps — v1.1
- STT errors return structured JSON for all failure modes — v1.1
- Whisper CLI backend preserved as default STT backend — v1.1
- STT backend selection via --backend local|whisper flag — v1.1
- STT model selection via --model flag with resolution chain — v1.1
- Transcript enriched with confidence scores and word-level timestamps — v1.1

### Active

(No active requirements — planning next milestone)

### Out of Scope

- Multi-user support — single learner, single machine
- Web UI or API server for the engine itself — CLI only, consumed by agent
- Custom voice training or voice cloning — use stock voices
- Replacing Azure Speech for pronunciation assessment — separate from TTS
- Running on non-Apple Silicon hardware — M4 Mac Mini is the target
- Offline mode — real-time local processing is core value

## Context

- **Target hardware:** Mac Mini M4 (Apple Silicon, unified memory, Metal/MPS)
- **TTS model:** Qwen3-TTS 1.7B via mlx-audio (MLX framework, ~2-3 GB RAM)
- **STT model:** mlx-community/whisper-large-v3-turbo via mlx-audio
- **Integration:** mlx-audio provides OpenAI-compatible HTTP server at localhost for both TTS and STT
- **Cold start:** Model loading takes 10-30s, so server must persist between calls
- **Agent flow:** Agent starts server at session begin, makes TTS/STT calls during lesson, stops at end
- **Current codebase:** ~6,138 LOC Go, well-structured with pluggable backends
- **Shipped:** v1.0 Local TTS (2026-03-01), v1.1 Local STT (2026-03-02)

## Constraints

- **Platform:** macOS ARM64 (Apple Silicon M4) — mlx-audio requires this
- **Architecture:** Two processes required — Go CLI (stateless) + mlx-audio server (persistent, holds model)
- **Communication:** HTTP to localhost (mlx-audio exposes OpenAI-compatible `/v1/audio/speech` and `/v1/audio/transcriptions`)
- **Technology:** Go for CLI, Python/MLX for TTS/STT server (managed as subprocess)
- **Backwards compatibility:** Existing CLI contract must not break — new commands additive only

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Keep Go codebase, don't rewrite | 90% of code unrelated to TTS; rewrite wastes 2-3 weeks | Good |
| Qwen3-TTS via mlx-audio | Best Chinese quality + native Apple Silicon optimization | Good |
| HTTP localhost for TTS/STT communication | mlx-audio already provides OpenAI-compatible server | Good |
| Server lifecycle in CLI | Agent needs single interface; no launchd/systemd knowledge needed | Good |
| Keep edge-tts as fallback | Zero-cost fallback when internet available | Good |
| LocalTTSError as value type | errors.As matches directly, simpler than pointer receivers | Good |
| Multipart form upload for STT | OpenAI-compatible API requires file upload, not JSON body | Good |
| STT backend default "whisper" | Preserves existing behavior, no breaking change | Good |
| ProcessVoice dual-backend routing | sttBackend resolved independently from TTS backend | Good |
| Additive transcript metadata | Confidence/word fields omitted when absent, no breaking change | Good |

---
*Last updated: 2026-03-02 after v1.1 milestone*
