---
phase: 02-local-tts-backend
plan: 01
subsystem: audio
tags: [mlx-audio, tts, local, http-client, ffmpeg, wav-conversion]

# Dependency graph
requires:
  - phase: 01-server-lifecycle/01
    provides: StartServer, state files (PID/port/model), readPIDFile, readPortFile, readModelFile, processAlive, cleanStateFiles
  - phase: 01-server-lifecycle/02
    provides: StopServer, ServerStatus
provides:
  - audio.LocalTTS() function for local TTS via mlx-audio HTTP API
  - CLI routing of --backend local to LocalTTS
affects: [02-local-tts-backend]

# Tech tracking
tech-stack:
  added: []
  patterns: [http-post-tts, temp-wav-cleanup, server-liveness-check-before-request]

key-files:
  created: [internal/xuezh/audio/local.go]
  modified: [internal/xuezh/cli/cli.go]

key-decisions:
  - "LocalTTS checks server liveness (PID + processAlive) before HTTP request to give clear error on stale PID"
  - "Default voice is Vivian (mlx-audio Qwen3-TTS default) rather than XiaoxiaoNeural (edge-tts default)"
  - "HTTP timeout set to 120s for long text synthesis"
  - "Error type changed from INVALID_ARGUMENT to BACKEND_FAILED for non-tool/non-process errors in local backend path"

patterns-established:
  - "LocalTTS reads server state files (port, model, pid) to validate server is running before HTTP request"
  - "Temp WAV file pattern: .local-tts-{uuid}.wav in same directory as output, cleaned up via defer"
  - "JSON body format for mlx-audio: {model, input, voice} posted to /v1/audio/speech"

# Metrics
duration: 5min
completed: 2026-03-01
---

# Plan 02-01: Local TTS Synthesis Summary

**HTTP POST TTS synthesis to mlx-audio server with server liveness validation, WAV-to-target format conversion via ffmpeg, and CLI --backend local routing**

## Performance

- **Duration:** 5 min
- **Started:** 2026-03-01
- **Completed:** 2026-03-01
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- LocalTTS function in audio/local.go: validates server state, POSTs to /v1/audio/speech, writes temp WAV, converts via ffmpeg
- Server liveness pre-check via PID file + processAlive before HTTP request
- Default voice "Vivian" for mlx-audio (vs "XiaoxiaoNeural" for edge-tts)
- CLI handler routes --backend local to LocalTTS, edge-tts path unchanged
- Proper error envelopes: TOOL_MISSING for ffmpeg, BACKEND_FAILED for server/HTTP errors, ProcessFailedError for ffmpeg failures

## Task Commits

Each task was committed atomically:

1. **Task 1: Create LocalTTS function in audio/local.go** - `550f888` (feat)
2. **Task 2: Route "local" backend through LocalTTS in CLI handler** - `05999d8` (feat)

## Files Created/Modified
- `internal/xuezh/audio/local.go` - LocalTTS function: HTTP POST to mlx-audio, temp WAV handling, ffmpeg conversion, artifact construction
- `internal/xuezh/cli/cli.go` - Branch in runAudioTTS: resolvedBackend=="local" routes to LocalTTS, error type changed to BACKEND_FAILED for plain errors

## Decisions Made
- LocalTTS validates server liveness (PID file + processAlive) before making HTTP request, giving clear "server not running" or "stale PID" errors instead of connection refused
- Default voice is "Vivian" (mlx-audio's Qwen3-TTS voice) when empty string passed, while edge-tts default remains "XiaoxiaoNeural"
- Error handling for non-tool/non-process errors changed from INVALID_ARGUMENT to BACKEND_FAILED in the local backend path, since server/HTTP errors are backend failures not argument errors
- 120s HTTP timeout accommodates long text synthesis on slower hardware

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- LocalTTS function ready for use by agent via `audio tts --backend local --text "..." --out path.ogg`
- Server must be started first via `audio server start` (Phase 1)
- Ready for Plan 02-02 (if exists)

---
*Phase: 02-local-tts-backend*
*Completed: 2026-03-01*
