---
phase: 04-local-stt-client
plan: 01
subsystem: audio
tags: [stt, whisper, multipart, http-client, mlx-audio]

# Dependency graph
requires:
  - phase: 02-local-tts-backend
    provides: LocalTTS pattern (error types, HTTP client, server liveness checks, classifiers)
  - phase: 01-server-lifecycle
    provides: server state files (pid, port, model), processAlive, cleanStateFiles
provides:
  - LocalSTT() function for local STT via mlx-audio server
  - LocalSTTError type with 7 classified failure modes
  - STT-specific HTTP error and response classifiers
affects: [05-stt-backend-selection, 06-stt-configuration]

# Tech tracking
tech-stack:
  added: [mime/multipart]
  patterns: [multipart form upload, verbose_json response parsing]

key-files:
  created: [internal/xuezh/audio/local_stt.go]
  modified: []

key-decisions:
  - "LocalSTTError mirrors LocalTTSError as value type with same Error() format"
  - "Multipart form upload with file/model/language/response_format fields"
  - "verbose_json response format extracts text + segments + duration via extractTranscript"
  - "Default STT model: mlx-community/whisper-large-v3-turbo (configurable in Phase 6)"
  - "HTTP 415 added to unsupported_format classification (not present in TTS classifier)"
  - "model_error reason added for HTTP 500 with model/load/not-found keywords"

patterns-established:
  - "STT error classification: same pattern as TTS but with STT-specific HTTP status/body matching"
  - "Multipart form upload: CreateFormFile + WriteField for OpenAI-compatible transcription API"

# Metrics
duration: 4min
completed: 2026-03-02
---

# Phase 4, Plan 01: Local STT Client Summary

**LocalSTT HTTP client with multipart upload to mlx-audio /v1/audio/transcriptions, structured error classification, and verbose_json transcript parsing**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-02
- **Completed:** 2026-03-02
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments
- LocalSTTError value type with 7 classified failure modes (server_down, stale_pid, oom, timeout, port_conflict, unsupported_format, model_error)
- STT-specific HTTP error and response classifiers mirroring TTS patterns
- LocalSTT function with multipart form upload, server liveness validation, and response parsing
- Transcript extraction reuses extractTranscript with added duration field

## Task Commits

Each task was committed atomically:

1. **Task 1: Create LocalSTTError type and STT error classification** - `4c9e131` (feat)
2. **Task 2: Create LocalSTT function with multipart HTTP client and response parsing** - `8d38ce7` (feat)

## Files Created/Modified
- `internal/xuezh/audio/local_stt.go` - LocalSTTError type, classifySTTHTTPError, classifySTTHTTPResponse, LocalSTT function, defaultSTTModel constant

## Decisions Made
- LocalSTTError mirrors LocalTTSError exactly as a value type (struct, not pointer) with same Error() format
- Added HTTP 415 (Unsupported Media Type) to unsupported_format classification since STT deals with file uploads
- Added "model_error" reason for HTTP 500 responses containing model/load/not-found keywords
- Default STT model set to "mlx-community/whisper-large-v3-turbo" as constant; Phase 6 will make configurable
- Language hardcoded to "zh" matching project scope (Chinese STT)
- Response format "verbose_json" used to get segments with timestamps and duration

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- LocalSTT function ready for integration into STT backend selection (Phase 5)
- Error types ready for CLI error envelope handling
- Transcript format compatible with existing extractTranscript and ProcessVoice pipelines

---
*Phase: 04-local-stt-client*
*Completed: 2026-03-02*
