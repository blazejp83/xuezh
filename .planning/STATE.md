# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-28)

**Core value:** Reliable, low-latency Chinese TTS that works locally on Apple Silicon without internet dependency, controllable via CLI by an AI agent.
**Current focus:** Phase 2 — Local TTS Backend (plan 02 complete)

## Current Position

Phase: 2 of 3 (Local TTS Backend) — IN PROGRESS
Plan: 02 complete
Status: Plan 02-02 complete (LocalTTSError with 6 classified failure modes + CLI error envelopes), ready for Plan 02-03
Last activity: 2026-03-01 — Plan 02-02 completed (error classification for local TTS backend)

Progress: ████░░░░░░ ~40%

## Performance Metrics

**Velocity:**
- Total plans completed: 4
- Average duration: 5 min
- Total execution time: ~0.35 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-server-lifecycle | 2 | 12 min | 6 min |
| 02-local-tts-backend | 2 | 9 min | 4.5 min |

**Recent Trend:**
- Last 5 plans: 8 min, 4 min, 5 min, 4 min
- Trend: stable/improving

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Used os/exec.Command directly in audio/server.go (not process.RunChecked) for non-blocking subprocess spawn
- Error messages use structured format BACKEND_FAILED:reason:detail for CLI layer extraction
- Server state tracked via plain-text files in workspace/server/ (pid, port, model)
- Port/model resolution chain: CLI flag -> config -> env -> default (port 8921, model Qwen3-TTS-12Hz-0.6B-Base-8bit)
- Three-phase shutdown: SIGTERM to process group -> 5s wait -> SIGKILL to process group -> 2s wait
- Uptime calculated from PID file mtime (approximate but sufficient)
- Stop/status return exit 0 for "no server running" (success, not error)
- LocalTTS validates server liveness (PID + processAlive) before HTTP request
- Default voice "Vivian" for local backend (vs "XiaoxiaoNeural" for edge-tts)
- 120s HTTP timeout for long text synthesis on local backend
- Non-tool/non-process errors in local backend path use BACKEND_FAILED (not INVALID_ARGUMENT)
- LocalTTSError is a value type (struct, not pointer) — errors.As matches directly
- HTTP error classification inspects body content case-insensitively for OOM/text_too_long keywords
- Connection refused + live PID = port_conflict; connection refused + dead PID = stale_pid
- Connection reset classified as oom (process killed during request)
- Response body capped at 1KB for body_preview in error details
- Unclassified HTTP errors default to server_down with full details

### Pending Todos

None yet.

### Blockers/Concerns

None yet.

## Session Continuity

Last session: 2026-03-01
Stopped at: Phase 2, Plan 02 complete (LocalTTSError with classified failure modes + CLI error envelopes)
Resume file: .planning/phases/02-local-tts-backend/02-02-SUMMARY.md
