---
phase: 03-polish-integration
plan: 02
subsystem: cli, audio
tags: [tts, instruct, speech-style, local-tts, qwen3-tts]

# Dependency graph
requires:
  - phase: 03-polish-integration
    plan: 01
    provides: Backend validation guard, LocalTTS with voice default, available_voices metadata
provides:
  - CLI --instruct flag for speech style control (local backend only)
  - LocalTTS passes instruct to mlx-audio HTTP body when non-empty
  - instruct included in response envelope when used
affects: [03-polish-integration]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - json.Marshal for conditional HTTP body field handling (replaces fmt.Sprintf)
    - Backend-specific flag validation (--instruct rejected for non-local backends)

key-files:
  created: []
  modified:
    - internal/xuezh/cli/cli.go
    - internal/xuezh/audio/local.go

key-decisions:
  - "json.Marshal replaces fmt.Sprintf for HTTP body construction — cleaner conditional field handling"
  - "--instruct with non-local backend returns INVALID_ARGUMENT, not silently ignored"
  - "instruct field only included in HTTP body and response envelope when non-empty"

patterns-established:
  - "Backend-specific flag validation: reject flags that don't apply to the resolved backend"
  - "Conditional HTTP body fields via json.Marshal map instead of string formatting"

# Metrics
duration: 4min
completed: 2026-03-01
---

# Plan 03-02: Speech Style Control via --instruct Parameter Summary

**CLI --instruct flag for Qwen3-TTS speech style control, wired through LocalTTS to mlx-audio server HTTP body**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-01
- **Completed:** 2026-03-01
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- CLI accepts --instruct flag for speech style instructions (e.g. "speak slowly", "excited tone")
- --instruct validated as local-backend-only; rejected with INVALID_ARGUMENT for edge-tts
- LocalTTS HTTP body switched from fmt.Sprintf to json.Marshal for clean conditional instruct field
- Response envelope includes instruct value when used, for agent visibility

## Task Commits

Each task was committed atomically:

1. **Task 2: Wire instruct parameter into LocalTTS HTTP body** - `80d9db5` (feat)
2. **Task 1: Add --instruct flag to CLI with backend validation** - `21c7969` (feat)

_Note: Task 2 committed first because LocalTTS signature change is required before CLI can call it with the new parameter._

## Files Created/Modified
- `internal/xuezh/cli/cli.go` - Added --instruct flag, backend validation for instruct, updated LocalTTS call
- `internal/xuezh/audio/local.go` - Updated LocalTTS signature, replaced fmt.Sprintf with json.Marshal body, added instruct to response data

## Decisions Made
- Replaced fmt.Sprintf body construction with json.Marshal for cleaner conditional field handling
- --instruct with non-local backend returns INVALID_ARGUMENT error (explicit rejection, not silent ignore)
- instruct field conditionally included in both HTTP body and response envelope (omitted when empty)

## Deviations from Plan

None - plan executed exactly as written. Commit order was reversed (Task 2 before Task 1) to maintain compile-time correctness across commits.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- TTS-05 (speech style control) complete
- --instruct parameter end-to-end from CLI through to mlx-audio HTTP body
- Ready for plan 03-03 if additional polish tasks remain

---
*Phase: 03-polish-integration*
*Completed: 2026-03-01*
