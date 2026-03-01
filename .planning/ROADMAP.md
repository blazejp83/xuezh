# Roadmap: xuezh — Local Chinese Learning Engine

## Overview

Add local Qwen3-TTS support to xuezh in three phases: first make the mlx-audio server reliably manageable via CLI, then wire up TTS synthesis through it, then polish with fallback support, voice options, and style control. Only ~920 LOC in the audio module needs modification.

## Domain Expertise

None

## Phases

- [x] **Phase 1: Server Lifecycle** - Start/stop/status commands with PID tracking and health polling
- [x] **Phase 2: Local TTS Backend** - HTTP client calling mlx-audio server with ffmpeg conversion
- [ ] **Phase 3: Polish & Integration** - Edge-tts fallback, backend selection, voices, style control

## Phase Details

### Phase 1: Server Lifecycle
**Goal**: Agent can start, stop, and query the mlx-audio TTS server via CLI commands with reliable process management
**Depends on**: Nothing (first phase)
**Requirements**: SRVR-01, SRVR-02, SRVR-03, SRVR-04, SRVR-05
**Research**: Unlikely (process management well-documented, Ollama reference implementation)
**Plans**: TBD

Plans:
- [x] 01-01: Server spawn with process group isolation, PID/port tracking, health polling, warm-up
- [x] 01-02: Server stop with clean process group shutdown, status command

### Phase 2: Local TTS Backend
**Goal**: Agent can generate Chinese speech via HTTP calls to running mlx-audio server
**Depends on**: Phase 1
**Requirements**: TTS-01, TTS-02, TTS-03, TTS-04
**Research**: Unlikely (HTTP client + existing ffmpeg pipeline)
**Plans**: TBD

Plans:
- [x] 02-01: HTTP client calling /v1/audio/speech, voice selection, ffmpeg conversion
- [x] 02-02: Structured JSON error handling for all 6 failure modes

### Phase 3: Polish & Integration
**Goal**: Edge-tts fallback preserved, backend selection flag, instruct parameter, dialect voices
**Depends on**: Phase 2
**Requirements**: TTS-05, TTS-06, BKND-01, BKND-02
**Research**: Unlikely (config and fallback logic already in codebase)
**Plans**: TBD

Plans:
- [x] 03-01: Backend selection flag, edge-tts backward compatibility
- [ ] 03-02: Instruct parameter for speech style, dialect voice support

## Progress

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Server Lifecycle | 2/2 | Complete | 2026-02-28 |
| 2. Local TTS Backend | 2/2 | Complete | 2026-03-01 |
| 3. Polish & Integration | 1/2 | In progress | - |
