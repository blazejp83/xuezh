# Requirements: xuezh — Local Chinese Learning Engine

**Defined:** 2026-02-28
**Core Value:** Reliable, low-latency Chinese TTS that works locally on Apple Silicon without internet dependency, controllable via CLI by an AI agent.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Server Lifecycle

- [x] **SRVR-01**: Agent can start local TTS server via `audio server start` with configurable port and model
- [x] **SRVR-02**: Agent can stop local TTS server via `audio server stop` with clean process group shutdown
- [x] **SRVR-03**: Agent can check server status via `audio server status` (running/stopped, PID, port)
- [x] **SRVR-04**: Server start polls for readiness before returning success (or timeout error after 120s)
- [x] **SRVR-05**: Server start runs warm-up TTS request before declaring ready

### TTS Synthesis

- [x] **TTS-01**: Agent can generate Chinese speech from text via HTTP POST to local server
- [x] **TTS-02**: Agent can select from available Chinese voices (default: Vivian)
- [x] **TTS-03**: Audio output converts from server WAV to ogg/mp3 via existing ffmpeg pipeline
- [x] **TTS-04**: TTS errors return structured JSON for all failure modes (server down, crash/stale PID, OOM, timeout, port conflict, text too long)
- [x] **TTS-05**: Agent can control speech style via optional instruct parameter
- [x] **TTS-06**: Agent can use dialect voices (Dylan for Beijing, Eric for Sichuan)

### Backend Management

- [x] **BKND-01**: Edge-tts backend continues to work unchanged
- [x] **BKND-02**: Agent selects backend explicitly per TTS call (`--backend local` vs `--backend edge-tts`)

## v1.1 Requirements

Requirements for local STT milestone. Replaces whisper CLI with mlx-audio HTTP transcription.

### Local STT Client

- [x] **STT-01**: Agent can transcribe Chinese audio via HTTP POST to `/v1/audio/transcriptions` on running mlx-audio server
- [x] **STT-02**: Transcription response parsed into structured transcript (text, segments with timestamps)
- [x] **STT-03**: STT errors return structured JSON for failure modes (server down, timeout, unsupported format, model error)

### STT Backend Selection

- [x] **STT-04**: Whisper CLI backend continues to work unchanged
- [x] **STT-05**: Agent selects STT backend explicitly (`--backend local` vs `--backend whisper`)

### STT Polish

- [x] **STT-06**: Agent can select transcription model for local STT backend
- [x] **STT-07**: Transcription response includes confidence scores and word-level timestamps when available

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Streaming & Advanced

- **STRM-01**: TTS output streams progressively for reduced perceived latency
- **AUTO-01**: Server auto-starts on first local TTS request if not running
- **MODL-01**: Support multiple TTS models via model parameter
- **VDSG-01**: Create custom voices from text descriptions via VoiceDesign model

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Voice cloning | Privacy concerns, quality variability, needs Base model variant |
| Automatic backend failover | Masks server failures, inconsistent voice between backends |
| Multiple simultaneous servers | 2-3 GB RAM each, no demonstrated need |
| HTTPS/TLS for localhost | No security benefit, breaks debugging |
| Web UI | Consumer is CLI/agent, unnecessary attack surface |
| Real-time conversation mode | SRS pattern is discrete clips, not continuous |

## Traceability

Which phases cover which requirements. Updated by create-roadmap.

| Requirement | Phase | Status |
|-------------|-------|--------|
| SRVR-01 | Phase 1 | Complete |
| SRVR-02 | Phase 1 | Complete |
| SRVR-03 | Phase 1 | Complete |
| SRVR-04 | Phase 1 | Complete |
| SRVR-05 | Phase 1 | Complete |
| TTS-01 | Phase 2 | Complete |
| TTS-02 | Phase 2 | Complete |
| TTS-03 | Phase 2 | Complete |
| TTS-04 | Phase 2 | Complete |
| TTS-05 | Phase 3 | Complete |
| TTS-06 | Phase 3 | Complete |
| BKND-01 | Phase 3 | Complete |
| BKND-02 | Phase 3 | Complete |

| STT-01 | Phase 4 | Complete |
| STT-02 | Phase 4 | Complete |
| STT-03 | Phase 4 | Complete |
| STT-04 | Phase 5 | Complete |
| STT-05 | Phase 5 | Complete |
| STT-06 | Phase 6 | Complete |
| STT-07 | Phase 6 | Complete |

**Coverage:**
- v1.0 requirements: 13 total (all complete)
- v1.1 requirements: 7 total (all complete)
- Mapped to phases: 20
- Unmapped: 0 ✓

---
*Requirements defined: 2026-02-28*
*Last updated: 2026-03-02 after v1.1 milestone completion*
