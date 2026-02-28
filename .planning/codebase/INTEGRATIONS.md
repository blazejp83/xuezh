# External Integrations

**Analysis Date:** 2026-02-28

## APIs & External Services

**Azure Speech Services (Cognitive Services):**
- Purpose: Pronunciation assessment + speech-to-text
- Integration: REST API (hand-rolled HTTP, no SDK)
- Location: `internal/xuezh/audio/azure.go`
- Auth: API key from file (`XUEZH_AZURE_SPEECH_KEY_FILE`), region from env/config (`XUEZH_AZURE_SPEECH_REGION`)
- Features: Pronunciation grading (HundredMark scale), phoneme-level assessment, comprehensive dimensions
- Language: zh-CN (Mandarin Chinese)
- Config section: `[azure.speech]` in `config.toml`

**edge-tts (Microsoft Edge TTS):**
- Purpose: Text-to-speech (neural Chinese voices)
- Integration: Python subprocess call
- Location: `internal/xuezh/audio/audio.go` (line 85, `buildTTSCommand()`)
- Auth: None required (unofficial free API)
- Voices: zh-CN-XiaoxiaoNeural (default), YunxiNeural, YunyangNeural
- Risk: Unofficial API, could break without notice

## Data Storage

**Databases:**
- SQLite - Primary data store (single-user, local)
  - Connection: `sql.Open("sqlite3", dbPath)`
  - Driver: `github.com/mattn/go-sqlite3` (CGO)
  - Location: `~/.clawdbot/workspace/xuezh/xuezh.db` (or `XUEZH_DB_PATH`)
  - Migrations: Embedded SQL in `migrations/` (4 migrations, `embed.FS`)
  - Init: `internal/xuezh/db/db.go`

**File Storage:**
- Local workspace artifact tree
  - Location: `~/.clawdbot/workspace/xuezh/artifacts/YYYY/MM/DD/`
  - Types: audio (WAV/OGG/MP3), JSON (transcripts, assessments)
  - Managed by: `internal/xuezh/paths/paths.go`
  - GC: `internal/xuezh/retention/retention.go` (configurable retention windows)

**Caching:**
- Content cache in SQLite `generated_content` table
- Managed by: `internal/xuezh/content/content.go`

## External CLI Tools

**ffmpeg:**
- Purpose: Audio format conversion (WAV 16kHz PCM, OGG Opus, MP3)
- Integration: Subprocess via `process.RunChecked()`
- Location: `internal/xuezh/audio/audio.go` (lines 67-83)
- Required: Yes (audio pipeline depends on it)

**whisper:**
- Purpose: Local speech-to-text fallback
- Integration: Subprocess
- Location: `internal/xuezh/audio/audio.go` (line 90)
- Model: tiny, language zh
- Required: Optional (Azure STT is primary)

**yt-dlp:**
- Purpose: Media download
- Integration: Available in dev environment (`devenv.nix`)
- Required: Optional

## Authentication & Identity

**Auth Provider:**
- No user authentication (single-user local tool)
- Azure API key stored in file, path configured via `XUEZH_AZURE_SPEECH_KEY_FILE`

## Monitoring & Observability

**Error Tracking:**
- None (local CLI tool)
- Errors returned as JSON envelopes

**Analytics:**
- None

**Logs:**
- stdout only (structured JSON envelopes)

## CI/CD & Deployment

**CI Pipeline:**
- GitHub Actions (`.github/workflows/ci.yml`)
- Trigger: Push to main + pull requests
- Steps: Checkout -> Install Nix + devenv -> Run `scripts/check.sh`

**Build:**
- Nix flakes (`flake.nix`) - `pkgs.buildGoModule`
- Output: Single binary at `/bin/xuezh`

**Deployment:**
- Local installation via Nix
- No cloud deployment

## Environment Configuration

**Development:**
- Required: Nix package manager, direnv
- Dev shell: `devenv.nix` provides all tools (Go, ffmpeg, edge-tts, etc.)
- Secrets: agenix-managed Azure key loaded in dev shell
- Config: `~/.config/xuezh/config.toml`

**Production:**
- Same as development (local CLI tool)
- Azure key file required for pronunciation assessment
- Internet required for edge-tts and Azure Speech API

## Infrastructure as Code

**Azure Speech Resource:**
- Tool: OpenTofu (Terraform-compatible)
- Location: `infra/azure/speech/`
- Manages: Azure Cognitive Services resource creation
- Region: West Europe (primary)

---

*Integration audit: 2026-02-28*
*Update when adding/removing external services*
