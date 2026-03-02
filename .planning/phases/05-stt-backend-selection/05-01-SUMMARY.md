---
phase: 05-stt-backend-selection
plan: 01
subsystem: cli, audio
tags: [stt, backend-selection, whisper, local-stt, cli-routing]

# Dependency graph
requires:
  - phase: 04-local-stt-client
    provides: LocalSTT function, LocalSTTError type, STT error classifiers
  - phase: 03-polish-integration
    provides: Backend validation guard pattern, resolveAudioBackend, TTS handler pattern
provides:
  - audio stt CLI command with --in, --backend flags and backend resolution chain
  - ProcessVoice STT routing through resolved backend (local or whisper)
  - LocalSTTError handling in both stt and process-voice CLI handlers
affects: [06-stt-configuration]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - STT backend validation guard (allowlist: local, whisper)
    - STT backend resolution chain (flag > config > env > default "whisper")

key-files:
  created: []
  modified:
    - internal/xuezh/cli/cli.go
    - internal/xuezh/audio/audio.go

key-decisions:
  - "STT backend default is 'whisper' (preserves existing behavior, STT-04)"
  - "Backend validation uses allowlist (local, whisper) matching TTS pattern"
  - "ProcessVoice sttBackend resolved independently from process_voice_backend"
  - "LocalSTTError handling added to both audio stt and audio process-voice handlers"

patterns-established:
  - "STT backend selection: same resolution chain as TTS (flag > config > env > default)"
  - "ProcessVoice dual-backend: process_voice_backend for assessment, stt_backend for transcription"

# Metrics
duration: 4min
completed: 2026-03-02
---

# Plan 05-01: STT Backend Selection Summary

**`audio stt` CLI command with --backend flag (local|whisper) and ProcessVoice STT routing through resolved backend**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-02
- **Completed:** 2026-03-02
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- `audio stt --in <path> [--backend local|whisper] --json` command registered in CLI router
- Backend resolution chain: flag > config (stt_backend) > env (XUEZH_AUDIO_STT_BACKEND) > default "whisper"
- ProcessVoice now routes STT through resolved backend instead of hardcoding whisper
- LocalSTTError, ToolMissingError, and ProcessFailedError all emit structured error envelopes

## Task Commits

Each task was committed atomically:

1. **Task 1: Add audio stt CLI subcommand with backend selection** - `6c3f30a` (feat)
2. **Task 2: Update ProcessVoice to route STT through resolved backend** - `8d11532` (feat)

## Files Created/Modified
- `internal/xuezh/cli/cli.go` - Added `case "stt"` to runAudio switch, created runAudioStt handler, resolved sttBackend in runAudioProcessVoice, added LocalSTTError handling to process-voice error chain
- `internal/xuezh/audio/audio.go` - Added sttBackend parameter to ProcessVoice, route to LocalSTT when sttBackend is "local"

## Decisions Made
- STT backend default is "whisper" to preserve existing behavior (STT-04 requirement)
- Backend validation uses allowlist approach (only "local" and "whisper" accepted) matching TTS pattern
- ProcessVoice resolves sttBackend independently via XUEZH_AUDIO_STT_BACKEND env var and stt_backend config key
- LocalSTTError handling placed between AzureSpeechError and ToolMissingError in process-voice error chain

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- STT backend selection complete (STT-04, STT-05)
- Ready for Phase 6: STT configuration (model selection, language options)
- All backend paths compile and route correctly

---
*Phase: 05-stt-backend-selection*
*Completed: 2026-03-02*
