# Coding Conventions

**Analysis Date:** 2026-02-28

## Naming Patterns

**Files:**
- `snake_case.go` for all Go source files
- One primary file per package (e.g., `srs/srs.go`, `db/db.go`)
- Additional files by domain (e.g., `audio/azure.go` alongside `audio/audio.go`)

**Functions:**
- PascalCase for exported: `InitDB()`, `NowUTC()`, `ListDueItems()`, `ProcessVoice()`
- camelCase for unexported: `intervalDays()`, `dueExpr()`, `expandHome()`, `relativeTo()`
- Verb-first naming: `Build*`, `List*`, `Parse*`, `Resolve*`, `Upsert*`

**Variables:**
- camelCase: `dbPath`, `itemID`, `dueAt`, `backend`
- Unexported package-level: `voiceAliases`, `supportedFormats`, `allowedContentTypes`
- No underscore prefix for private

**Types:**
- PascalCase for exported structs: `OKEnvelope`, `AudioResult`, `AzureSpeechError`
- No interface prefix (no `I` prefix)

**IDs:**
- Prefix format: 2-3 letter code + underscore + payload
- `w_`, `g_`, `c_`, `ev_`, `ct_`, `ar_`, `ds_`

## Code Style

**Formatting:**
- Standard `gofmt` (tabs, Go defaults)
- No custom `.editorconfig` or formatter config
- No explicit line length limit (reasonable lengths observed)

**Linting:**
- No `.golangci.yml` detected
- CI runs `go test ./...` via `scripts/check.sh`
- Standard Go vet implied

## Import Organization

**Order:**
1. Standard library (`database/sql`, `fmt`, `os`, `time`)
2. Blank imports for side effects (`_ "github.com/mattn/go-sqlite3"`)
3. External packages (`github.com/oklog/ulid/v2`, `github.com/pelletier/go-toml/v2`)
4. Internal packages (`github.com/joshp123/xuezh/internal/xuezh/*`)

**Grouping:**
- Blank line between standard library and external
- Internal imports grouped with externals

## Error Handling

**Patterns:**
- Functions return `(T, error)` tuples (idiomatic Go)
- Error wrapping with `fmt.Errorf("context: %w", err)`
- Immediate post-call checking with early returns
- Custom error types for typed discrimination at CLI boundary

**Error Types (from `internal/xuezh/errors/errors.go`):**
- `BACKEND_FAILED` - service/tool failure
- `AUTH_FAILED` - authentication issues
- `INVALID_ARGUMENT` - CLI/validation errors
- `NOT_IMPLEMENTED` - unimplemented features
- `NOT_FOUND` - resource not found
- `QUOTA_EXCEEDED` - service quota
- `TOOL_MISSING` - CLI tool not on PATH

**Custom Error Structs:**
- `ToolMissingError` in `process/process.go`
- `ProcessFailedError` in `process/process.go`
- `AzureSpeechError` with Kind field in `audio/azure.go`
- CLI uses `errors.As()` to discriminate and map to envelope error types

## Logging

**Framework:**
- No logging framework (stdout-only JSON envelopes)
- All output is structured JSON via `jsonio.Dumps()`
- Errors emitted as JSON error envelopes, never to stderr

## Comments

**When to Comment:**
- Minimal inline comments — code is self-documenting
- Standard Go package comments above `package` statements
- Some exported function comments

**TODO Comments:**
- No TODO/FIXME comments found in Go codebase
- Placeholder logic marked with descriptive values (e.g., `"local_v0_placeholder"`)

## Function Design

**Size:**
- Most functions under 50 lines
- CLI handler functions are larger (flag parsing + business logic)
- `cli.go` is the largest file at 1116 lines

**Parameters:**
- Multiple positional parameters (Go convention)
- Pointer types for optional values (e.g., `*string`, `*int`)

**Return Values:**
- `(T, error)` for fallible operations
- Early returns on error (guard clauses)
- Named returns not used

## Module Design

**Exports:**
- Named exports only (Go convention)
- Each package exposes focused public API
- Internal helpers unexported

**Package Boundaries:**
- CLI imports all services (centripetal)
- Services import utilities only (no cross-service imports)
- Utilities have no service dependencies
- No circular imports

## Development Methodology

**RGR Workflow (from AGENTS.md):**
- Red: Enable/extend tests first
- Green: Implement minimum code to pass
- Refactor: Clean up without breaking

**Commit Discipline:**
- One ticket = one atomic commit
- Message format: `T-XX: <ticket title>`
- No fixup/WIP commits

**Contract-First:**
- CLI contract in `docs/cli-contract.md` is authoritative
- JSON schemas in `schemas/` define output shape
- BDD scenarios in `specs/bdd/` for behavior verification

---

*Convention analysis: 2026-02-28*
*Update when patterns change*
