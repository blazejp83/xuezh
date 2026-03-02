# Roadmap: xuezh — Local Chinese Learning Engine

## Overview

Add local mlx-audio support to xuezh for both TTS and STT on Apple Silicon. v1.0 shipped local TTS via Qwen3-TTS. v1.1 adds local STT by consuming the same mlx-audio server's transcription endpoint, replacing the whisper CLI subprocess with GPU-accelerated HTTP calls.

## Domain Expertise

None

## Milestones

- ✅ **v1.0 Local TTS** — Phases 1-3 (shipped 2026-03-01)
- 🚧 **v1.1 Local STT** — Phases 4-6 (in progress)

## Phases

<details>
<summary>✅ v1.0 Local TTS (Phases 1-3) — SHIPPED 2026-03-01</summary>

### Phase 1: Server Lifecycle
**Goal**: Agent can start, stop, and query the mlx-audio TTS server via CLI commands with reliable process management
**Depends on**: Nothing (first phase)
**Requirements**: SRVR-01, SRVR-02, SRVR-03, SRVR-04, SRVR-05

Plans:
- [x] 01-01: Server spawn with process group isolation, PID/port tracking, health polling, warm-up
- [x] 01-02: Server stop with clean process group shutdown, status command

### Phase 2: Local TTS Backend
**Goal**: Agent can generate Chinese speech via HTTP calls to running mlx-audio server
**Depends on**: Phase 1
**Requirements**: TTS-01, TTS-02, TTS-03, TTS-04

Plans:
- [x] 02-01: HTTP client calling /v1/audio/speech, voice selection, ffmpeg conversion
- [x] 02-02: Structured JSON error handling for all 6 failure modes

### Phase 3: Polish & Integration
**Goal**: Edge-tts fallback preserved, backend selection flag, instruct parameter, dialect voices
**Depends on**: Phase 2
**Requirements**: TTS-05, TTS-06, BKND-01, BKND-02

Plans:
- [x] 03-01: Backend selection flag, edge-tts backward compatibility
- [x] 03-02: Instruct parameter for speech style, dialect voice support

</details>

### 🚧 v1.1 Local STT (In Progress)

**Milestone Goal:** Replace whisper CLI subprocess with HTTP calls to existing mlx-audio server for GPU-accelerated transcription on Apple Silicon

#### Phase 4: Local STT Client
**Goal**: HTTP client calling /v1/audio/transcriptions on existing mlx-audio server, response parsing, error classification for STT failure modes
**Depends on**: v1.0 complete (server lifecycle already managed)
**Requirements**: STT-01, STT-02, STT-03
**Research**: Likely (mlx-audio /v1/audio/transcriptions API schema, supported models, multipart form format)
**Research topics**: mlx-audio transcription endpoint request/response format, model names, audio format requirements
**Plans**: TBD

Plans:
- [x] 04-01: LocalSTT HTTP client with multipart upload, response parsing, error classification

#### Phase 5: STT Backend Selection
**Goal**: Backend selection flag for STT (local vs whisper), backward compatibility with whisper CLI, process-voice pipeline update
**Depends on**: Phase 4
**Requirements**: STT-04, STT-05
**Research**: Unlikely (mirrors BKND-01/BKND-02 pattern from TTS)
**Plans**: TBD

Plans:
- [ ] 05-01: TBD (run /gsd:plan-phase 5 to break down)

#### Phase 6: STT Polish
**Goal**: Model selection, transcript format improvements, confidence/timestamp metadata
**Depends on**: Phase 5
**Requirements**: STT-06, STT-07
**Research**: Unlikely (internal patterns, metadata extraction)
**Plans**: TBD

Plans:
- [ ] 06-01: TBD (run /gsd:plan-phase 6 to break down)

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Server Lifecycle | v1.0 | 2/2 | Complete | 2026-02-28 |
| 2. Local TTS Backend | v1.0 | 2/2 | Complete | 2026-03-01 |
| 3. Polish & Integration | v1.0 | 2/2 | Complete | 2026-03-01 |
| 4. Local STT Client | v1.1 | 1/1 | Complete | 2026-03-02 |
| 5. STT Backend Selection | v1.1 | 0/? | Not started | - |
| 6. STT Polish | v1.1 | 0/? | Not started | - |
