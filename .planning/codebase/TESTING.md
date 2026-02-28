# Testing Patterns

**Analysis Date:** 2026-02-28

## Test Framework

**Runner:**
- Go built-in `testing` package
- No Go test files currently exist in the codebase (0 `*_test.go` files)

**External Test Infrastructure:**
- pytest-bdd for BDD executable specs (`specs/bdd/`)
- Python contract tests (`tests/contract/`)
- Contract sync validation (`tests/contract/test_contract_bdd_sync.py`)

**Run Commands:**
```bash
./scripts/check.sh                    # CI check (runs go test ./...)
go test ./...                         # All Go tests
```

## Test File Organization

**Location:**
- Go tests: co-located with source (`*_test.go` alongside `*.go`) — convention, but none exist yet
- BDD specs: `specs/bdd/*.feature`
- Contract tests: `tests/contract/*.py`

**Naming:**
- Go: `<module>_test.go` (expected convention)
- BDD: `*.feature` files
- Contract: `test_contract_*.py`

**Structure:**
```
internal/xuezh/
  srs/
    srs.go              # Source (no srs_test.go)
  audio/
    audio.go            # Source (no audio_test.go)
    azure.go            # Source (no azure_test.go)
  ...                   # Pattern repeats: 20 source files, 0 test files

specs/
  bdd/                  # BDD feature files
  test-strategy.md      # Testing pyramid strategy

tests/
  contract/             # Python-based contract validation
  conftest.py           # Pytest config (forbids skipped tests)
```

## Test Strategy (from `specs/test-strategy.md`)

**Testing Pyramid:**

**Unit Tests (fast, many):**
- JSON envelope shape
- Deterministic ordering helpers
- Schema validation
- Path safety (no traversal)
- Schedule transforms (pure functions)

**Integration Tests (CLI):**
- Invoking `xuezh` commands
- Exit code validation
- JSON envelope schema validation
- Bounded output behavior (`--max-bytes`, `--limit`)

**E2E Tests (flows):**
- Temp workspace + SQLite
- Dataset import -> snapshot -> review -> grade -> report

## Test Infrastructure

**Testability Hooks:**
- `XUEZH_TEST_NOW_ISO` env var overrides `clock.NowUTC()` for deterministic time
- `XUEZH_WORKSPACE_DIR` redirects workspace for test isolation
- `XUEZH_DB_PATH` redirects database location
- `XUEZH_RETENTION_*_DAYS` overrides GC retention windows

**Contract Enforcement:**
- JSON schemas in `schemas/` define output contracts (23 schemas)
- Contract tests validate CLI output matches schemas
- No skipped tests policy: `tests/conftest.py` forbids `@pytest.mark.skip` (use `xfail` instead)

## Coverage

**Requirements:**
- No enforced coverage target
- Zero Go test coverage currently (no test files)
- BDD and contract tests provide behavioral coverage

## Test Types

**Unit Tests:**
- Not implemented yet in Go
- Strategy calls for pure function testing (SRS scheduling, ID generation, path safety)

**Integration Tests:**
- Contract tests run CLI commands and validate JSON output
- CI: `.github/workflows/ci.yml` runs `scripts/check.sh`

**E2E Tests:**
- Strategy defined but not fully implemented
- Would use temp workspace + SQLite for full flow testing

## Mocking

**Not applicable yet** (no test files). Expected patterns:
- Interface-based mocking for database
- Environment variable overrides for time/paths
- Temp directories for workspace isolation

## Test Coverage Gaps

**Critical untested paths:**
- SRS scheduling logic (`internal/xuezh/srs/srs.go` - 229 lines)
- Audio processing (`internal/xuezh/audio/audio.go` - 626 lines)
- Azure Speech integration (`internal/xuezh/audio/azure.go` - 293 lines)
- Report generation (`internal/xuezh/reports/reports.go` - 609 lines)
- Database migrations (`internal/xuezh/db/db.go` - 132 lines)
- All 20 Go source files (4,677 lines total)

---

*Testing analysis: 2026-02-28*
*Update when test patterns change*
