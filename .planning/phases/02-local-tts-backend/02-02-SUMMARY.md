---
phase: 02-local-tts-backend
plan: 02
subsystem: audio
tags: [mlx-audio, tts, local, error-classification, error-envelope]

# Dependency graph
requires:
  - phase: 02-local-tts-backend/01
    provides: LocalTTS function, readPIDFile, readPortFile, readModelFile, processAlive, cleanStateFiles
provides:
  - audio.LocalTTSError type with 6 classified failure modes (server_down, stale_pid, oom, timeout, port_conflict, text_too_long)
  - CLI structured error envelopes with reason field for agent action routing
affects: [02-local-tts-backend]

# Tech tracking
tech-stack:
  added: []
  patterns: [classified-error-type, error-envelope-reason-field, http-error-classification]

key-files:
  created: []
  modified: [internal/xuezh/audio/local.go, internal/xuezh/cli/cli.go]

key-decisions:
  - "LocalTTSError is a value type (not pointer) in audio package — errors.As matches it directly"
  - "HTTP error classification uses body content inspection (case-insensitive) for OOM/text_too_long detection"
  - "Connection refused with live PID classified as port_conflict; with dead PID classified as stale_pid"
  - "Connection reset classified as oom (process killed during request)"
  - "Response body limited to 1KB for body_preview in error details"
  - "Unclassified HTTP errors default to server_down with full details"

patterns-established:
  - "LocalTTSError.Reason field provides agent-switchable failure mode classification"
  - "CLI merges LocalTTSError.Details into standard error envelope details map"
  - "LocalTTSError checked via errors.As before ToolMissingError/ProcessFailedError in CLI handler"

# Metrics
duration: 4min
completed: 2026-03-01
---

# Plan 02-02: Local TTS Error Classification Summary

**LocalTTSError type with 6 classified failure modes (server_down, stale_pid, oom, timeout, port_conflict, text_too_long) and CLI structured error envelopes with agent-actionable reason field**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-01
- **Completed:** 2026-03-01
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- LocalTTSError struct with Reason/Message/Details fields exported from audio package
- All 6 failure modes classified with distinct reasons and structured details
- HTTP error classification: timeout via net.Error, connection refused/reset via net.OpError, response body inspection for OOM/text_too_long
- CLI handler emits BACKEND_FAILED error envelopes with reason field merged from LocalTTSError.Details
- Agent can switch on details.reason to decide action (restart server, alert user, retry, split text)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create LocalTTSError type with 6 failure mode classifications** - `31e035a` (feat)
2. **Task 2: Wire LocalTTSError into CLI handler for structured JSON error envelopes** - `27d3085` (feat)

## Files Created/Modified
- `internal/xuezh/audio/local.go` - Added LocalTTSError type, classifyHTTPError/classifyHTTPResponse helpers, isTimeout/isConnectionRefused/isConnectionReset utilities, replaced plain errors with classified LocalTTSError returns
- `internal/xuezh/cli/cli.go` - Added errors.As check for audio.LocalTTSError in runAudioTTS, merges reason and details into BACKEND_FAILED error envelope

## Decisions Made
- LocalTTSError is a value type (struct, not pointer) so errors.As works with both value and pointer returns
- HTTP error classification inspects body content case-insensitively for keywords (memory, oom, killed, too long, too large, max length, exceeds)
- Connection refused + live PID = port_conflict; connection refused + dead PID = stale_pid (with auto-cleanup)
- Connection reset assumed to be OOM/crash (process killed during request)
- Response body capped at 1KB in body_preview detail field to avoid oversized error envelopes
- Unclassified HTTP errors default to server_down reason with full details as fallback

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- All 6 local TTS failure modes produce distinct classified errors with agent-actionable reason field
- Existing edge-tts error handling unchanged
- Ready for next plan in phase 02 (if exists)

---
*Phase: 02-local-tts-backend*
*Completed: 2026-03-01*
