# Project Research Summary

**Project:** xuezh — Local Chinese Learning Engine
**Domain:** CLI tool with local ML model integration (TTS on Apple Silicon)
**Researched:** 2026-02-28
**Confidence:** HIGH

## Executive Summary

Adding local Qwen3-TTS to xuezh is well-supported by the mlx-audio ecosystem. The package provides an OpenAI-compatible HTTP server (`POST /v1/audio/speech`) that runs natively on Apple Silicon via MLX. The Go CLI spawns this as a managed subprocess, polls for readiness, then makes HTTP calls — replacing the current `edge-tts` subprocess-per-call pattern.

The recommended model is `mlx-community/Qwen3-TTS-12Hz-0.6B-Base-8bit` (~2 GB, best quality/size ratio, 17.7k downloads). Chinese voices include Vivian, Serena, Uncle_Fu, Dylan (Beijing dialect), and Eric (Sichuan). Cold start is 15-40s (model loading + Metal shader JIT), so the server must persist across TTS calls within a session.

The biggest risks are: zombie process trees (Go must kill the entire process group, not just PID), MLX memory leaks during sustained inference (may need periodic restarts), and lazy model loading (server reports "ready" before model is actually loaded — must warm up with a throwaway TTS request).

## Key Findings

### Recommended Stack

**Core technologies:**
- `mlx-audio` v0.3.1 — OpenAI-compatible TTS server on Apple Silicon
- `mlx-community/Qwen3-TTS-12Hz-0.6B-Base-8bit` — best quality/size model for M4
- Python >= 3.10 (arm64 native) — runtime for mlx-audio in isolated venv
- Go 1.22 — existing CLI, no changes to language

**Critical dependency pins:** `transformers == 5.0.0rc3` and `mlx-lm == 0.30.5` — requires isolated Python environment to avoid conflicts.

### Expected Features

**Must have (table stakes):**
- TTS endpoint via HTTP to localhost (`POST /v1/audio/speech`)
- Server lifecycle CLI (`audio server start/stop/status`)
- Health check + model warm-up before declaring ready
- Voice selection (Vivian default for Chinese learners)
- Error handling for 6 failure modes (server down, OOM, timeout, port conflict, text too long, stale PID)
- edge-tts preserved as fallback backend

**Should have (differentiators):**
- Emotion/style control via `instruct` parameter ("speak slowly and clearly")
- Dialect voices (Dylan=Beijing, Eric=Sichuan) at zero implementation cost

**Defer (v2+):**
- Streaming TTS output (minimal benefit for short sentences)
- Voice cloning (out of scope)

### Architecture Approach

Two-process sidecar pattern: Go CLI is stateless, mlx-audio server holds model in memory. Communication via HTTP on localhost. PID + port tracked in workspace files. Process group isolation (`Setpgid: true`) ensures clean shutdown of entire process tree.

**Major components:**
1. **Server Manager** (Go) — spawn, health check, PID tracking, shutdown
2. **TTS Backend** (Go) — HTTP client calling `/v1/audio/speech`, response handling
3. **mlx-audio Server** (Python) — model hosting, inference, API serving

### Critical Pitfalls

1. **Zombie processes** — Must use process groups (`Setpgid: true`, kill with negative PID). Uvicorn has known orphan process bugs.
2. **MLX memory leaks** — Confirmed in GitHub issues. Monitor RSS, restart server if exceeds threshold.
3. **Lazy model loading** — Server reports ready before model loads. Must warm up with throwaway TTS request.
4. **Port conflicts** — macOS AirPlay uses port 5000. Use non-standard port (8921). Check port availability before spawn.
5. **Go pipe deadlock** — `cmd.Wait()` hangs if child inherits pipes. Use `WaitDelay` (Go 1.20+), redirect stdout/stderr to files.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Server Lifecycle Management
**Rationale:** Everything depends on being able to start/stop the mlx-audio server reliably
**Delivers:** `audio server start/stop/status` commands with PID tracking, process group management, health polling
**Addresses:** F-02 (lifecycle), F-03 (health check), F-04 (warm-up)
**Avoids:** Zombie process pitfall, pipe deadlock pitfall

### Phase 2: Local TTS Backend
**Rationale:** Once server is manageable, wire up TTS calls through it
**Delivers:** `audio tts --backend mlx-audio` making HTTP calls to running server, with ffmpeg conversion
**Addresses:** F-01 (TTS endpoint), F-06 (audio format), F-07 (error handling)
**Uses:** Existing audio pipeline (ffmpeg conversion already exists)

### Phase 3: Polish & Fallback
**Rationale:** Harden edge cases, ensure edge-tts still works, add voice/style options
**Delivers:** Edge-tts fallback preserved, voice selection, emotion/style control, config documentation
**Addresses:** F-05 (voices), F-08 (fallback), D-01 (emotion control)
**Avoids:** Memory leak pitfall (add monitoring/restart)

### Phase Ordering Rationale

- Phase 1 first because no TTS calls work without a running server
- Phase 2 depends on Phase 1 (needs server running to test)
- Phase 3 is polish that can happen after core flow works end-to-end
- Total scope is small (~920 LOC in audio module + new server management)

### Research Flags

Phases with standard patterns (skip research-phase):
- **Phase 1:** Process management is well-documented (Ollama reference implementation)
- **Phase 2:** HTTP client calls are straightforward Go
- **Phase 3:** Configuration and fallback logic already exists in codebase

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | mlx-audio PyPI, GitHub, and model hub all verified |
| Features | HIGH | OpenAI API contract well-documented, mlx-audio API confirmed |
| Architecture | HIGH | Ollama provides proven reference for Go + ML server pattern |
| Pitfalls | HIGH | GitHub issues with reproduction steps, Go process management well-documented |

**Overall confidence:** HIGH

### Gaps to Address

- mlx-audio has no documented `/health` endpoint — must verify `GET /v1/models` works as readiness probe during Phase 1
- Memory leak severity on M4 specifically is unverified — monitor during Phase 3
- `transformers == 5.0.0rc3` is a release candidate pin — may need updating if mlx-audio releases a new version

## Sources

### Primary (HIGH confidence)
- [mlx-audio GitHub](https://github.com/Blaizzy/mlx-audio) — API surface, server commands, model support
- [mlx-audio PyPI](https://pypi.org/project/mlx-audio/) — v0.3.1, dependency pins
- [OpenAI TTS API](https://platform.openai.com/docs/api-reference/audio/createSpeech) — API contract
- [MLX GitHub](https://github.com/ml-explore/mlx) — Apple Silicon ML framework
- [Ollama llm/server.go](https://github.com/ollama/ollama) — Go subprocess management reference

### Secondary (MEDIUM confidence)
- [MLX memory leak issues](https://github.com/ml-explore/mlx-examples/issues/1124) — confirmed but fix status unclear
- [Uvicorn orphan process bug](https://github.com/Kludex/uvicorn/issues/2289) — workaround via process groups
- [Go process group killing](https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773) — well-documented pattern

---
*Research completed: 2026-02-28*
*Ready for roadmap: yes*
