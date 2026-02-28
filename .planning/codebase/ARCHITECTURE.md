# Architecture

**Analysis Date:** 2026-02-28

## Pattern Overview

**Overall:** Modular CLI Application with ZFC (Zero Framework Cognition) design

**Key Characteristics:**
- Single compiled binary with subcommands
- "Smart endpoint + dumb pipes" — engine handles mechanics, bot/LLM handles pedagogy
- All output wrapped in JSON envelopes for machine consumption
- Pluggable audio backends via config hierarchy
- SQLite persistence, file-based artifact storage

## Layers

**CLI Dispatch Layer:**
- Purpose: Parse user input, route to handlers, format JSON output
- Contains: Command definitions, flag parsing, envelope wrapping
- Location: `internal/xuezh/cli/cli.go` (1116 lines, ~70 handler functions)
- Entry: `cmd/xuezh-go/main.go` (12 lines, calls `cli.Run()`)
- Depends on: All service packages
- Used by: External callers (Clawdbot agent)

**Service/Domain Layer:**
- Purpose: Core business logic for each domain
- Contains: SRS scheduling, audio processing, reports, events, datasets, content, snapshots
- Locations:
  - `internal/xuezh/srs/srs.go` - Spaced repetition (SM2/Leitner)
  - `internal/xuezh/audio/audio.go` - TTS, format conversion, voice processing
  - `internal/xuezh/audio/azure.go` - Azure Speech REST API
  - `internal/xuezh/reports/reports.go` - HSK/mastery/due reports
  - `internal/xuezh/events/events.go` - Event logging
  - `internal/xuezh/datasets/datasets.go` - CSV import
  - `internal/xuezh/content/content.go` - Content caching
  - `internal/xuezh/snapshot/snapshot.go` - Learning state snapshots
- Depends on: Utility layer, database
- Used by: CLI dispatch layer

**Utility/Infrastructure Layer:**
- Purpose: Shared helpers with no business logic
- Contains: Config, paths, clock, IDs, JSON I/O, process execution, error types
- Locations:
  - `internal/xuezh/config/config.go` - TOML config (singleton, sync.Once)
  - `internal/xuezh/paths/paths.go` - Workspace paths, sandbox enforcement
  - `internal/xuezh/clock/clock.go` - UTC time, testable via env override
  - `internal/xuezh/ids/ids.go` - Deterministic ID generation (SHA1 + ULID)
  - `internal/xuezh/jsonio/jsonio.go` - JSON serialization
  - `internal/xuezh/process/process.go` - Tool execution, PATH lookup
  - `internal/xuezh/errors/errors.go` - Known error type registry
  - `internal/xuezh/envelope/envelope.go` - Response envelope structure
  - `internal/xuezh/retention/retention.go` - GC retention policies
- Depends on: Standard library only
- Used by: Service layer

**Data/Persistence Layer:**
- Purpose: SQLite database + embedded migrations
- Contains: DB init, migration runner, schema
- Location: `internal/xuezh/db/db.go`, `migrations/*.sql`
- Depends on: go-sqlite3 driver
- Used by: Service layer (srs, reports, events, datasets, content)

## Data Flow

**CLI Command Execution (e.g., `review start`):**

1. User/agent invokes: `xuezh review start --limit 10 --json`
2. `main.go` calls `cli.Run(args)`
3. CLI dispatcher parses command, routes to `runReviewStart()`
4. Handler parses flags, calls `srs.ListDueItems()`
5. Service queries SQLite, returns structured result
6. Handler wraps in `envelope.OK()` with data + limits
7. `jsonio.Dumps()` serializes to pretty JSON
8. Output to stdout, exit code 0

**Audio TTS Flow:**

1. `xuezh audio tts --text "你好" --backend edge-tts --json`
2. Backend resolved: CLI flag > config > env > default
3. `audio.TTSAudio()` spawns `edge-tts` subprocess -> temp MP3
4. `ffmpeg` converts MP3 to target format (OGG/WAV)
5. Artifact stored in `artifacts/YYYY/MM/DD/` workspace tree
6. Return envelope with artifact metadata (path, MIME, bytes)

**State Management:**
- SQLite database at `~/.clawdbot/workspace/xuezh/xuezh.db`
- File artifacts in dated workspace directories
- No in-memory state between CLI invocations (stateless per call)

## Key Abstractions

**JSON Envelope:**
- Purpose: Uniform response wrapper for all commands
- Location: `internal/xuezh/envelope/envelope.go`
- Pattern: `OKEnvelope{ok, schema_version, command, data, artifacts, truncated, limits}`
- Error variant: `ErrorEnvelope{ok:false, error{type, message, details}}`

**Service Result:**
- Purpose: Structured return from domain services
- Pattern: `{Data map[string]any, Artifacts []Artifact, Truncated bool, Limits map[string]any}`
- Services never emit JSON directly; CLI wraps in envelope

**ID Scheme:**
- Purpose: Deterministic, non-colliding identifiers
- Prefixes: `w_` (word), `g_` (grammar), `c_` (char), `ev_` (event/ULID), `ct_` (content), `ar_` (artifact), `ds_` (dataset)
- Location: `internal/xuezh/ids/ids.go`

**Backend Resolution:**
- Purpose: Pluggable audio processing backends
- Pattern: CLI flag > config section > global config > env var > default
- Location: `internal/xuezh/cli/cli.go` `resolveAudioBackend()`

## Entry Points

**Binary Entry:**
- Location: `cmd/xuezh-go/main.go`
- Triggers: CLI invocation (`xuezh <command>`)
- Responsibilities: Parse args, call `cli.Run()`, exit with code

**CLI Router:**
- Location: `internal/xuezh/cli/cli.go`
- Triggers: Dispatched from main
- Responsibilities: Route 11 top-level commands, parse flags, call services, emit envelopes

## Error Handling

**Strategy:** Typed error discrimination at CLI boundary

**Patterns:**
- Services return `(T, error)` tuples (idiomatic Go)
- Errors wrapped with `fmt.Errorf("context: %w", err)`
- CLI uses `errors.As()` to match custom types (`AzureSpeechError`, `ToolMissingError`, `ProcessFailedError`)
- 7 known error types: `BACKEND_FAILED`, `AUTH_FAILED`, `INVALID_ARGUMENT`, `NOT_IMPLEMENTED`, `NOT_FOUND`, `QUOTA_EXCEEDED`, `TOOL_MISSING`
- All errors wrapped in `ErrorEnvelope` with structured details

## Cross-Cutting Concerns

**Logging:**
- stdout only (JSON envelopes)
- No structured logging framework
- Errors emitted as JSON error envelopes to stdout

**Validation:**
- Flag parsing in CLI handlers
- Path safety via `paths.ResolveInWorkspace()` (prevents directory traversal)
- ID validation via regex patterns in `ids/ids.go`
- Error type validation via `errors.AssertKnown()`

**Testability:**
- `XUEZH_TEST_NOW_ISO` env var overrides current time
- `XUEZH_WORKSPACE_DIR` / `XUEZH_DB_PATH` redirect storage
- `XUEZH_RETENTION_*_DAYS` override GC windows

---

*Architecture analysis: 2026-02-28*
*Update when major patterns change*
