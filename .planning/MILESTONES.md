# Project Milestones: xuezh — Local Chinese Learning Engine

## v1.1 Local STT (Shipped: 2026-03-02)

**Delivered:** Local GPU-accelerated STT via mlx-audio server, replacing whisper CLI subprocess with HTTP transcription calls on Apple Silicon.

**Phases completed:** 4-6 (3 plans total)

**Key accomplishments:**

- LocalSTT HTTP client with multipart form upload to mlx-audio /v1/audio/transcriptions
- 7-mode STT error classification mirroring TTS patterns
- `audio stt` CLI command with `--backend local|whisper` backend selection
- ProcessVoice dual-backend routing for independent STT transcription
- STT model selection via `--model` flag with resolution chain
- Enriched transcript metadata with confidence scores, word-level timestamps, duration

**Stats:**

- 10 files created/modified (949 insertions, 30 deletions)
- 6,138 lines of Go (total codebase)
- 3 phases, 3 plans, 6 tasks
- 1 day from start to ship

**Git range:** `feat(04-01)` → `feat(06-01)`

**What's next:** TBD — streaming TTS, auto-start server, or new direction

---

## v1.0 Local TTS (Shipped: 2026-03-01)

**Delivered:** Local Chinese TTS via Qwen3-TTS on mlx-audio server with full server lifecycle management and backend selection.

**Phases completed:** 1-3 (6 plans total)

**Key accomplishments:**

- Server lifecycle management (start/stop/status) with process group isolation
- LocalTTS HTTP client with voice selection, ffmpeg conversion, structured errors
- Backend selection (`--backend local|edge-tts`) with backward compatibility
- Instruct parameter for speech style control and dialect voice support

**Stats:**

- ~920 LOC modified in audio module
- 4,600+ lines of Go (codebase at ship time)
- 3 phases, 6 plans
- 2 days from start to ship (2026-02-28 → 2026-03-01)

**Git range:** `feat(01-01)` → `feat(03-02)`

---
