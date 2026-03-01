---
phase: 03-polish-integration
plan: 01
subsystem: cli, audio
tags: [tts, backend-selection, voice-discovery, edge-tts, local-tts]

# Dependency graph
requires:
  - phase: 02-local-tts-backend
    provides: LocalTTS function with voice default and error classification
provides:
  - Backend validation rejecting invalid TTS backends with INVALID_ARGUMENT
  - Backend-aware voice defaults (XiaoxiaoNeural for edge-tts, empty/Vivian for local)
  - available_voices metadata in local TTS response for agent discoverability
affects: [03-polish-integration]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - Backend validation guard before routing logic
    - available_voices metadata for agent voice discovery

key-files:
  created: []
  modified:
    - internal/xuezh/cli/cli.go
    - internal/xuezh/audio/local.go

key-decisions:
  - "Voice flag defaults to empty string — backend-specific defaults applied after validation"
  - "Backend validation uses allowlist (local, edge-tts) not blocklist"
  - "available_voices is informational metadata only — no client-side voice validation"

patterns-established:
  - "Backend validation guard: validate resolvedBackend before any routing logic"
  - "available_voices metadata pattern: include discoverable options in response data"

# Metrics
duration: 4min
completed: 2026-03-01
---

# Plan 03-01: Backend Selection & Voice Discovery Summary

**Backend-aware voice defaults, backend validation guard, and available_voices metadata for local TTS agent discoverability**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-01
- **Completed:** 2026-03-01
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- Voice flag now defaults to empty string; backend-specific voice applied after resolveAudioBackend
- Invalid TTS backends rejected with INVALID_ARGUMENT error envelope before any routing
- Local TTS response includes available_voices list (Vivian, Dylan, Eric) for agent discovery

## Task Commits

Each task was committed atomically:

1. **Task 1: Backend-aware voice default and backend validation** - `ad9d8ed` (feat)
2. **Task 2: Add available_voices metadata to local TTS response** - `0a535fe` (feat)

## Files Created/Modified
- `internal/xuezh/cli/cli.go` - Voice default changed to empty, backend validation guard added, backend-specific voice default applied
- `internal/xuezh/audio/local.go` - localVoices package variable added, available_voices included in response data map

## Decisions Made
- Voice flag defaults to empty string rather than "XiaoxiaoNeural" so local backend does not receive an edge-tts-specific voice name
- Backend validation uses allowlist approach (only "local" and "edge-tts" accepted)
- available_voices is for agent discoverability only; server-side voice validation remains authoritative

## Deviations from Plan

None - plan executed exactly as written

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Backend selection and voice discovery complete (BKND-01, BKND-02, TTS-06)
- Ready for plan 03-02: instruct parameter for speech style and dialect voice support

---
*Phase: 03-polish-integration*
*Completed: 2026-03-01*
