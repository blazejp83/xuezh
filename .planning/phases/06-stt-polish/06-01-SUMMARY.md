---
phase: 06-stt-polish
plan: 01
subsystem: audio
tags: [whisper, stt, mlx-audio, cli]

# Dependency graph
requires:
  - phase: 05-stt-backend-selection
    provides: STT backend routing (local vs whisper) in CLI and ProcessVoice
provides:
  - --model flag for STT model selection with resolution chain
  - sttModel threading through ProcessVoice to LocalSTT
  - Enriched transcript metadata (confidence scores, word-level timestamps, duration)
affects: []

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Model resolution chain: flag > config (stt_model) > env (XUEZH_AUDIO_STT_MODEL) > default"
    - "Additive metadata extraction: confidence/word fields only present when source provides them"

key-files:
  created: []
  modified:
    - internal/xuezh/cli/cli.go
    - internal/xuezh/audio/local_stt.go
    - internal/xuezh/audio/audio.go

key-decisions:
  - "Model resolution uses simple inline chain (not resolveAudioBackend which has global fallback)"
  - "DefaultSTTModel exported from local_stt.go as safety net; CLI uses own resolution chain"
  - "extractTranscript handles duration extraction — removed duplicate logic from LocalSTT"
  - "All new transcript fields are additive (omitted when absent from source data)"

patterns-established:
  - "STT model resolution: flag > config stt_model > env XUEZH_AUDIO_STT_MODEL > default mlx-community/whisper-large-v3-turbo"

# Metrics
duration: 4min
completed: 2026-03-02
---

# Phase 6: STT Polish Summary

**--model flag with resolution chain for STT backend selection, enriched transcripts with confidence scores and word-level timestamps**

## Performance

- **Duration:** 4 min
- **Started:** 2026-03-02
- **Completed:** 2026-03-02
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- STT model selectable via `audio stt --model` flag with resolution chain (flag > config > env > default)
- ProcessVoice accepts and threads sttModel to LocalSTT when using local backend
- extractTranscript enriched with avg_logprob, no_speech_prob, compression_ratio per segment
- Word-level timestamps (word, start, end, probability) extracted per segment when available
- Duration extracted at top level by extractTranscript, benefiting both local and whisper backends

## Task Commits

Each task was committed atomically:

1. **Task 1: Add --model flag and thread model through ProcessVoice** - `6a93d76` (feat)
2. **Task 2: Enrich transcript with confidence scores and word-level timestamps** - `e31b67a` (feat)

## Files Created/Modified
- `internal/xuezh/cli/cli.go` - Added --model flag to runAudioStt, STT model resolution in runAudioProcessVoice, updated ProcessVoice call signature
- `internal/xuezh/audio/local_stt.go` - Exported DefaultSTTModel constant, removed redundant duration extraction
- `internal/xuezh/audio/audio.go` - Added sttModel param to ProcessVoice, enriched extractTranscript with confidence/word/duration metadata

## Decisions Made
- Model resolution uses simple inline chain (flag > config > env > default), not resolveAudioBackend which has global backend fallback logic
- DefaultSTTModel exported as safety net for programmatic LocalSTT callers; CLI maintains its own chain
- extractTranscript now handles duration extraction itself, so removed duplicate logic from LocalSTT
- All new transcript fields are strictly additive -- omitted when source data doesn't include them

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Removed redundant duration extraction from LocalSTT**
- **Found during:** Task 2 (enriching extractTranscript)
- **Issue:** LocalSTT had explicit duration extraction that would duplicate the new extractTranscript logic
- **Fix:** Removed the redundant `if duration, ok := raw["duration"]; ok` block from local_stt.go
- **Files modified:** internal/xuezh/audio/local_stt.go
- **Verification:** go build ./... and go vet ./... pass
- **Committed in:** e31b67a (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary to avoid duplicate duration in transcript output. No scope creep.

## Issues Encountered
None

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- v1.1 Local STT milestone complete: all STT requirements (STT-01 through STT-07) implemented
- No blockers or concerns

---
*Phase: 06-stt-polish*
*Completed: 2026-03-02*
