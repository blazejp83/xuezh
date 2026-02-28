# Codebase Structure

**Analysis Date:** 2026-02-28

## Directory Layout

```
xuezh/
├── cmd/xuezh-go/          # Binary entry point
│   └── main.go            # 12 lines, calls cli.Run()
├── internal/xuezh/        # All application code (20 packages)
│   ├── cli/               # Command dispatcher (1116 lines)
│   ├── audio/             # TTS, STT, format conversion, Azure
│   ├── srs/               # Spaced repetition scheduling
│   ├── reports/           # HSK, mastery, due reports
│   ├── datasets/          # CSV dataset import
│   ├── events/            # Event logging
│   ├── content/           # Content caching
│   ├── snapshot/          # Learning state snapshots
│   ├── db/                # SQLite init & migrations
│   ├── envelope/          # JSON response wrapper
│   ├── errors/            # Error type registry
│   ├── config/            # TOML config loading
│   ├── paths/             # Workspace paths & safety
│   ├── clock/             # Time utilities
│   ├── ids/               # ID generation & validation
│   ├── jsonio/            # JSON serialization
│   ├── process/           # External tool execution
│   ├── retention/         # GC retention policies
│   └── migrations/        # Migration embed loader
├── migrations/            # SQL schema migrations
│   ├── 0001_initial.sql
│   ├── 0002_add_review_due.sql
│   ├── 0003_add_events.sql
│   └── 0004_add_srs_vectors.sql
├── schemas/               # JSON schemas (23 files, one per command)
├── datasets/              # Bundled HSK vocabulary data
│   └── ivankra-hsk30/
├── specs/                 # BDD specs & test strategy
│   ├── bdd/              # Feature files
│   └── test-strategy.md
├── tests/                 # Python contract tests
│   └── contract/
├── docs/                  # Documentation
│   ├── cli-contract.md   # Authoritative CLI spec
│   └── reference/        # ZFC philosophy, oracle prompt
├── skills/                # Bot skill definitions
│   └── xuezh/            # SKILL.md for Clawdbot
├── infra/                 # Infrastructure as Code
│   └── azure/speech/     # OpenTofu for Azure Speech
├── scripts/               # Build/check scripts
│   └── check.sh
├── go.mod                 # Go module definition
├── go.sum                 # Dependency checksums
├── flake.nix              # Nix build definition
├── devenv.nix             # Dev environment (Nix)
├── .envrc                 # direnv activation
├── .github/workflows/     # CI pipeline
│   └── ci.yml
├── AGENTS.md              # Development guidelines
└── README.md              # Project overview
```

## Directory Purposes

**cmd/xuezh-go/**
- Purpose: Binary entry point
- Contains: Single `main.go` (12 lines)
- Key files: `main.go` — parses args, calls `cli.Run()`, exits

**internal/xuezh/**
- Purpose: All application packages (Go internal convention)
- Contains: 20 packages, each single-file
- Subdirectories: One per domain/utility

**migrations/**
- Purpose: SQLite schema migrations (embedded via `embed.FS`)
- Contains: SQL files numbered 0001-0004
- Key files: `0001_initial.sql` (base schema, 11 tables)

**schemas/**
- Purpose: JSON Schema contracts for CLI output
- Contains: 23 schema files (one per command output)
- Key files: `envelope.ok.schema.json`, `envelope.err.schema.json`

**datasets/ivankra-hsk30/**
- Purpose: Bundled HSK 1-6 vocabulary and grammar
- Contains: CSV files + SOURCE.txt attribution

**specs/**
- Purpose: BDD specifications and test strategy
- Contains: Feature files, test strategy document

**infra/azure/speech/**
- Purpose: Azure Speech resource provisioning
- Contains: OpenTofu (Terraform) configuration

**skills/xuezh/**
- Purpose: Skill definition for Clawdbot integration
- Contains: SKILL.md describing how bot uses xuezh CLI

## Key File Locations

**Entry Points:**
- `cmd/xuezh-go/main.go` - Binary entry
- `internal/xuezh/cli/cli.go` - Command dispatcher

**Configuration:**
- `go.mod` - Go module definition
- `flake.nix` - Nix build configuration
- `devenv.nix` - Development environment
- `.envrc` - direnv activation

**Core Logic:**
- `internal/xuezh/srs/srs.go` - SRS scheduling
- `internal/xuezh/audio/audio.go` - Audio processing
- `internal/xuezh/audio/azure.go` - Azure Speech integration
- `internal/xuezh/reports/reports.go` - Report generation

**Testing:**
- `tests/contract/` - Python contract tests
- `specs/bdd/` - BDD feature files
- `scripts/check.sh` - CI check script

**Documentation:**
- `docs/cli-contract.md` - Authoritative CLI specification
- `AGENTS.md` - Development workflow guidelines
- `README.md` - Project overview

## Naming Conventions

**Files:**
- `snake_case.go` for Go source files
- `NNNN_description.sql` for migrations (zero-padded sequence)
- `command.subcommand.schema.json` for JSON schemas

**Directories:**
- lowercase single-word for Go packages (`srs`, `audio`, `cli`, `db`)
- kebab-case for non-Go directories (`ivankra-hsk30`)

**Special Patterns:**
- One `.go` file per package (most packages are single-file)
- `azure.go` as secondary file in `audio/` package

## Where to Add New Code

**New CLI Command:**
- Handler: `internal/xuezh/cli/cli.go` (add to dispatcher + handler function)
- Schema: `schemas/<command>.<subcommand>.schema.json`
- Contract: Update `docs/cli-contract.md`

**New Service/Domain Package:**
- Implementation: `internal/xuezh/<name>/<name>.go`
- Tests: `internal/xuezh/<name>/<name>_test.go`
- Wire into CLI: Import in `internal/xuezh/cli/cli.go`

**New Audio Backend:**
- Implementation: `internal/xuezh/audio/<backend>.go`
- Backend resolution: Update `resolveAudioBackend()` in `cli.go`
- Config: Add `[<backend>]` section support in config

**New Migration:**
- SQL: `migrations/NNNN_description.sql`
- Automatically picked up by embed.FS loader

## Special Directories

**schemas/**
- Purpose: JSON Schema contracts for every CLI command output
- Source: Hand-written, authoritative
- Committed: Yes

**datasets/ivankra-hsk30/**
- Purpose: Bundled reference data for HSK import
- Source: External (attributed in SOURCE.txt)
- Committed: Yes

---

*Structure analysis: 2026-02-28*
*Update when directory structure changes*
