# Technology Stack

**Analysis Date:** 2026-02-28

## Languages

**Primary:**
- Go 1.22 - All application code (`go.mod`)

**Secondary:**
- Python 3.13 - edge-tts TTS engine (`devenv.nix`)
- SQL - Database migrations (`migrations/*.sql`)
- Nix - Development environment and build (`flake.nix`, `devenv.nix`)

## Runtime

**Environment:**
- Go 1.22 (compiled binary, CGO required for SQLite)
- Python 3.13 (runtime dependency for edge-tts subprocess)
- No web server (CLI tool only)

**Package Manager:**
- Go modules (`go.mod`, `go.sum`)
- Nix flakes for dev environment and build (`flake.nix`, `flake.lock`)
- Lockfiles: `go.sum`, `flake.lock`, `devenv.lock`

## Frameworks

**Core:**
- None (vanilla Go CLI, no web framework)

**Testing:**
- Go built-in `testing` package (no test files yet)
- pytest-bdd for BDD specs (`specs/bdd/`)
- Contract tests in Python (`tests/contract/`)

**Build/Dev:**
- Nix flakes (`pkgs.buildGoModule` in `flake.nix`)
- devenv for development shell (`devenv.nix`)
- direnv for automatic shell activation (`.envrc`)

## Key Dependencies

**Critical (Go - from `go.mod`):**
- `github.com/mattn/go-sqlite3 v1.14.33` - SQLite driver (CGO, used in `internal/xuezh/db/db.go`)
- `github.com/pelletier/go-toml/v2 v2.2.2` - TOML config parsing (`internal/xuezh/config/config.go`)
- `github.com/oklog/ulid/v2 v2.1.1` - ULID generation for events (`internal/xuezh/ids/ids.go`)
- `github.com/google/uuid v1.6.0` - UUID generation (`internal/xuezh/audio/audio.go`)

**External CLI Tools (from `devenv.nix`):**
- ffmpeg - Audio format conversion (WAV/OGG/MP3)
- edge-tts - Text-to-speech (Microsoft neural voices)
- whisper - Local speech-to-text fallback
- yt-dlp - Media download

## Configuration

**Environment:**
- TOML config file: `~/.config/xuezh/config.toml` (XDG_CONFIG_HOME aware)
- Key env vars: `XUEZH_WORKSPACE_DIR`, `XUEZH_DB_PATH`, `XUEZH_TEST_NOW_ISO`
- Audio backend env vars: `XUEZH_AUDIO_BACKEND`, `XUEZH_AUDIO_TTS_BACKEND`
- Azure creds: `XUEZH_AZURE_SPEECH_KEY_FILE`, `XUEZH_AZURE_SPEECH_REGION`
- Secrets via agenix (Nix-based secret management)

**Build:**
- `go.mod` - Module definition
- `flake.nix` - Nix build configuration (produces single binary)

## Platform Requirements

**Development:**
- NixOS or any platform with Nix package manager
- direnv for automatic shell activation
- CGO toolchain (required by go-sqlite3)

**Production:**
- Single static binary from `nix build`
- Runtime deps: ffmpeg, edge-tts (Python), internet for edge-tts/Azure
- Workspace directory: `~/.clawdbot/workspace/xuezh/`

---

*Stack analysis: 2026-02-28*
*Update after major dependency changes*
