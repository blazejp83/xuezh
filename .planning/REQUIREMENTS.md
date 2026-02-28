# Requirements: xuezh — Local Chinese Learning Engine

**Defined:** 2026-02-28
**Core Value:** Reliable, low-latency Chinese TTS that works locally on Apple Silicon without internet dependency, controllable via CLI by an AI agent.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Server Lifecycle

- [ ] **SRVR-01**: Agent can start local TTS server via `audio server start` with configurable port and model
- [ ] **SRVR-02**: Agent can stop local TTS server via `audio server stop` with clean process group shutdown
- [ ] **SRVR-03**: Agent can check server status via `audio server status` (running/stopped, PID, port)
- [ ] **SRVR-04**: Server start polls for readiness before returning success (or timeout error after 120s)
- [ ] **SRVR-05**: Server start runs warm-up TTS request before declaring ready

### TTS Synthesis

- [ ] **TTS-01**: Agent can generate Chinese speech from text via HTTP POST to local server
- [ ] **TTS-02**: Agent can select from available Chinese voices (default: Vivian)
- [ ] **TTS-03**: Audio output converts from server WAV to ogg/mp3 via existing ffmpeg pipeline
- [ ] **TTS-04**: TTS errors return structured JSON for all failure modes (server down, crash/stale PID, OOM, timeout, port conflict, text too long)
- [ ] **TTS-05**: Agent can control speech style via optional instruct parameter
- [ ] **TTS-06**: Agent can use dialect voices (Dylan for Beijing, Eric for Sichuan)

### Backend Management

- [ ] **BKND-01**: Edge-tts backend continues to work unchanged
- [ ] **BKND-02**: Agent selects backend explicitly per TTS call (`--backend local` vs `--backend edge-tts`)

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
| SRVR-01 | Phase 1 | Pending |
| SRVR-02 | Phase 1 | Pending |
| SRVR-03 | Phase 1 | Pending |
| SRVR-04 | Phase 1 | Pending |
| SRVR-05 | Phase 1 | Pending |
| TTS-01 | Phase 2 | Pending |
| TTS-02 | Phase 2 | Pending |
| TTS-03 | Phase 2 | Pending |
| TTS-04 | Phase 2 | Pending |
| TTS-05 | Phase 3 | Pending |
| TTS-06 | Phase 3 | Pending |
| BKND-01 | Phase 3 | Pending |
| BKND-02 | Phase 3 | Pending |

**Coverage:**
- v1 requirements: 13 total
- Mapped to phases: 13
- Unmapped: 0 ✓

---
*Requirements defined: 2026-02-28*
*Last updated: 2026-02-28 after roadmap creation*
