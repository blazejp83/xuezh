---
phase: 01-server-lifecycle
plan: 01
subsystem: audio
tags: [mlx-audio, tts, subprocess, health-check, process-group]

# Dependency graph
requires: []
provides:
  - audio.StartServer() function for spawning and managing mlx-audio TTS server
  - ServerState type for tracking server lifecycle
  - CLI command `audio server start` with --port and --model flags
  - Stub commands `audio server stop` and `audio server status`
  - Server state files (PID, port, model) in workspace/server/
affects: [01-server-lifecycle]

# Tech tracking
tech-stack:
  added: []
  patterns: [subprocess-with-process-group, health-polling, state-file-tracking, warm-up-request]

key-files:
  created: [internal/xuezh/audio/server.go]
  modified: [internal/xuezh/cli/cli.go]

key-decisions:
  - "Use os/exec.Command directly in audio/server.go for non-blocking subprocess spawn (process.RunChecked is blocking)"
  - "Error strings encode structured info as BACKEND_FAILED:reason:message format for CLI layer to parse"

patterns-established:
  - "Server state tracked via plain-text files in workspace/server/ (pid, port, model, server.log)"
  - "Process group isolation via Setpgid for clean server lifecycle management"
  - "Health polling pattern: 250ms ticker with 120s deadline, checking both process liveness and HTTP endpoint"
  - "Warm-up request is non-fatal: failure logged but server still reported as ready"
  - "Port/model resolution chain: CLI flag -> config -> env -> hardcoded default"

# Metrics
duration: 8min
completed: 2026-02-28
---

# Plan 01-01: Server Start Command Summary

**mlx-audio server lifecycle with process group isolation, health polling, and warm-up TTS request via `audio server start` CLI command**

## Performance

- **Duration:** 8 min
- **Started:** 2026-02-28
- **Completed:** 2026-02-28
- **Tasks:** 2
- **Files modified:** 2

## Accomplishments
- ServerState struct and full server lifecycle management in audio/server.go
- Stale server detection and cleanup (PID check, HTTP probe, process group kill)
- Subprocess spawn with Setpgid for process group isolation
- Health polling with 250ms interval and 120s timeout, crash detection via log tail
- Non-fatal warm-up TTS request to pre-load model weights
- CLI handler with port/model resolution chain (flag -> config -> env -> default 8921)
- Stub commands for stop and status (NOT_IMPLEMENTED, ready for Plan 02)

## Task Commits

Each task was committed atomically:

1. **Task 1: Create audio/server.go with StartServer** - `28e04a9` (feat)
2. **Task 2: Add audio server start CLI handler** - `49d759b` (feat)

## Files Created/Modified
- `internal/xuezh/audio/server.go` - Server lifecycle: ServerState, StartServer, health polling, warm-up, stale cleanup
- `internal/xuezh/cli/cli.go` - CLI routing for audio server start/stop/status with flag parsing and envelope output

## Decisions Made
- Used os/exec.Command directly (not process.RunChecked) because StartServer needs non-blocking cmd.Start()
- Error messages use structured format (BACKEND_FAILED:reason:detail) so CLI layer can extract context

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- StartServer foundation complete, ready for Plan 02 (stop/status commands)
- State file infrastructure (PID, port, model) ready for stop/status to consume
- Server log file available at workspace/server/server.log for debugging

---
*Phase: 01-server-lifecycle*
*Completed: 2026-02-28*
