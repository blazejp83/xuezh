# Codebase Concerns

**Analysis Date:** 2026-02-28

## Tech Debt

**Duplicated utility functions across packages:**
- Issue: `expandHome()` implemented in 4 separate files
- Files: `internal/xuezh/audio/audio.go`, `internal/xuezh/audio/azure.go`, `internal/xuezh/datasets/datasets.go`, `internal/xuezh/content/content.go`
- Impact: Bug fixes must be applied in 4 places
- Fix approach: Consolidate into `internal/xuezh/paths/paths.go` (which already handles path utilities)

**Duplicated `relativeTo()` helper:**
- Issue: Same function in 2 files
- Files: `internal/xuezh/audio/audio.go`, `internal/xuezh/content/content.go`
- Fix approach: Move to `internal/xuezh/paths/paths.go`

**No database connection pooling:**
- Issue: Each function opens a new `sql.Open()` connection
- Files: `internal/xuezh/srs/srs.go`, `internal/xuezh/reports/reports.go`, `internal/xuezh/events/events.go`, `internal/xuezh/datasets/datasets.go`
- Impact: Unnecessary overhead on repeated calls
- Fix approach: Pass `*sql.DB` from CLI layer or use singleton connection

**Placeholder local assessment logic:**
- Issue: Local pronunciation assessment is only exact string matching
- File: `internal/xuezh/audio/audio.go` (lines 357-365, marked `"local_v0_placeholder"`)
- Impact: Useless without Azure backend
- Fix approach: Document as known limitation or implement phonetic comparison

## Known Bugs

No confirmed bugs detected. Codebase appears functional.

## Security Considerations

**Dynamic SQL construction in SRS queries:**
- Risk: `dueExpr()` builds SQL column names dynamically via `fmt.Sprintf`
- File: `internal/xuezh/srs/srs.go` (line 197)
- Current mitigation: `reviewType` validated against known values in `dueExpr()` (lines 48-56)
- Recommendations: Consider using explicit query variants instead of string interpolation

**Text passed to external processes without sanitization:**
- Risk: User-provided text passed directly to edge-tts subprocess
- File: `internal/xuezh/audio/audio.go` (line 85, `buildTTSCommand()`)
- Current mitigation: Go's `exec.Command()` doesn't use shell (arguments are separate)
- Recommendations: Low risk due to Go's exec model, but validate input length

**Ignored error on pronunciation storage:**
- Risk: Silent data loss if database write fails
- File: `internal/xuezh/audio/audio.go` (line 338, `_ = storePronunciationAttempt(...)`)
- Recommendations: Log or return the error

## Performance Bottlenecks

**Repeated database opens:**
- Problem: Each service function opens/closes SQLite independently
- Files: Throughout `internal/xuezh/srs/`, `reports/`, `events/`, `datasets/`
- Cause: No connection sharing between calls
- Improvement path: Connection injection from CLI layer

**Inefficient JSON parsing in reports:**
- Problem: Unmarshal to `map[string]any` then format-string conversion
- File: `internal/xuezh/reports/reports.go` (lines 294-310)
- Cause: Generic map access instead of typed struct
- Improvement path: Define typed structs for dataset payloads

## Fragile Areas

**edge-tts dependency:**
- File: `internal/xuezh/audio/audio.go`
- Why fragile: Unofficial API wrapping Microsoft Edge TTS service. Could break with any Microsoft update.
- Common failures: Network errors, API changes, voice removal
- Safe modification: Backend is already pluggable — adding alternative TTS is straightforward
- Test coverage: None

**CLI dispatcher size:**
- File: `internal/xuezh/cli/cli.go` (1116 lines)
- Why fragile: Single large file handling all command routing and flag parsing
- Common failures: Flag naming conflicts, handler wiring errors
- Safe modification: Follow existing patterns closely
- Test coverage: None (covered by contract tests externally)

## Dependencies at Risk

**edge-tts (Python package):**
- Risk: Unofficial reverse-engineering of Microsoft's free TTS API. No SLA, could break anytime.
- Impact: TTS fails completely if API changes
- Migration plan: Add local TTS backend (Qwen3-TTS via mlx-audio) — already planned

**go-sqlite3 (CGO dependency):**
- Risk: Requires CGO toolchain (C compiler). Blocks pure Go builds.
- Impact: Build failures on platforms without CGO (Alpine/musl)
- Migration plan: Consider `modernc.org/sqlite` (pure Go) if CGO becomes problematic

## Test Coverage Gaps

**All Go source code:**
- What's not tested: 20 source files, 4,677 lines, 0 test files
- Risk: Regressions undetectable
- Priority: High
- Difficulty: Testability hooks exist (env var overrides for time, paths, DB)

**SRS scheduling logic:**
- What's not tested: Interval calculation, grade-to-interval mapping, dual-modality scheduling
- File: `internal/xuezh/srs/srs.go` (229 lines)
- Risk: Incorrect review timing silently degrades learning
- Priority: Critical

**Audio pipeline:**
- What's not tested: TTS, STT, format conversion, Azure integration
- Files: `internal/xuezh/audio/audio.go` (626 lines), `azure.go` (293 lines)
- Risk: Audio failures break core learning flow
- Priority: High

## Missing Critical Features

**No local TTS fallback:**
- Problem: TTS requires internet (edge-tts) or fails
- Current workaround: None — offline use has no TTS
- Blocks: Offline learning sessions
- Implementation complexity: Medium (add mlx-audio/Qwen3-TTS backend)

---

*Concerns audit: 2026-02-28*
*Update as issues are fixed or new ones discovered*
