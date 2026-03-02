# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-28)

**Core value:** Reliable, low-latency Chinese TTS and STT that works locally on Apple Silicon without internet dependency, controllable via CLI by an AI agent.
**Current focus:** Milestone v1.1 Local STT — COMPLETE

## Current Position

Phase: 6 of 6 (STT Polish) — COMPLETE
Plan: 01 complete
Status: Phase 6 complete, v1.1 Local STT milestone complete
Last activity: 2026-03-02 — Phase 6 Plan 01 complete

Progress: ██████████ 100% (v1.0 complete, v1.1 complete)

## Performance Metrics

**Velocity:**
- Total plans completed: 9
- Average duration: 5 min
- Total execution time: ~0.66 hours

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-server-lifecycle | 2 | 12 min | 6 min |
| 02-local-tts-backend | 2 | 9 min | 4.5 min |
| 03-polish-integration | 2 | 8 min | 4 min |
| 04-local-stt-client | 1 | 4 min | 4 min |
| 05-stt-backend-selection | 1 | 4 min | 4 min |
| 06-stt-polish | 1 | 4 min | 4 min |

**Recent Trend:**
- Last 5 plans: 4 min, 4 min, 4 min, 4 min, 4 min
- Trend: stable

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
- Voice flag defaults to empty string — backend-specific defaults applied after validation
- Backend validation uses allowlist (local, edge-tts) not blocklist
- available_voices is informational metadata only — no client-side voice validation
- json.Marshal replaces fmt.Sprintf for LocalTTS HTTP body — cleaner conditional field handling
- --instruct with non-local backend returns INVALID_ARGUMENT, not silently ignored
- instruct field only included in HTTP body and response envelope when non-empty
- LocalSTTError mirrors LocalTTSError as value type with same BACKEND_FAILED:reason:message format
- HTTP 415 added to STT unsupported_format classification (file upload vs JSON body)
- model_error reason added for HTTP 500 with model/load/not-found keywords (STT-specific)
- Default STT model: mlx-community/whisper-large-v3-turbo (constant, configurable in Phase 6)
- STT language hardcoded to "zh" matching project scope
- verbose_json response format used for segments with timestamps and duration
- Multipart form upload for STT (file + model + language + response_format fields)
- STT backend default is "whisper" (preserves existing behavior, STT-04)
- STT backend validation uses allowlist (local, whisper) matching TTS pattern
- ProcessVoice sttBackend resolved independently from process_voice_backend
- LocalSTTError handling added to both audio stt and audio process-voice CLI handlers
- STT model resolution uses simple inline chain (not resolveAudioBackend which has global fallback)
- DefaultSTTModel exported from local_stt.go as safety net; CLI uses own resolution chain
- extractTranscript handles duration extraction (removed duplicate from LocalSTT)
- All transcript metadata fields (confidence, words, duration) are additive — omitted when absent

### Pending Todos

None.

### Blockers/Concerns

None.

### Roadmap Evolution

- Milestone v1.1 Local STT created: local STT via mlx-audio server, 3 phases (Phase 4-6)
- Milestone v1.1 Local STT complete: all STT requirements (STT-01 through STT-07) implemented

## Session Continuity

Last session: 2026-03-02
Stopped at: Phase 6 Plan 01 complete — v1.1 milestone complete
Resume file: None
