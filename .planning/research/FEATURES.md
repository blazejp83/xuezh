# Feature Research: Local TTS Server for xuezh

Research date: 2026-02-28
Domain: CLI tool managing a local Qwen3-TTS server (via mlx-audio) for Chinese language learning

---

## Table Stakes (Must Have for v1)

### F-01: OpenAI-Compatible TTS Endpoint

mlx-audio exposes `POST /v1/audio/speech` with JSON body parameters: `model`, `input` (text),
`voice`, and `speed`. Response is audio bytes (WAV by default). This matches the OpenAI API
contract where the same endpoint accepts `model`, `input`, `voice`, `response_format`, and
`speed` (0.25--4.0, default 1.0).

The Go CLI already knows how to call edge-tts and convert output. The new backend replaces
`edge-tts` subprocess calls with HTTP POST to localhost. No new protocol needed.

- Source: https://github.com/Blaizzy/mlx-audio (HIGH)
- Source: https://platform.openai.com/docs/api-reference/audio/createSpeech (HIGH)
- Source: https://pypi.org/project/mlx-audio/ (HIGH)

### F-02: Server Lifecycle Management (start/stop/status)

The agent needs three CLI commands:

- `audio server start` -- spawn `mlx_audio.server --host 127.0.0.1 --port <port>` as a
  background process, record PID, wait for health check to pass.
- `audio server stop` -- send SIGTERM to recorded PID, wait for clean exit, clean up PID file.
- `audio server status` -- report whether server is running, PID, port, model loaded.

FastAPI/Uvicorn handles SIGTERM natively: Uvicorn stops accepting new requests, finishes
in-flight requests, then exits. No custom shutdown handler needed in mlx-audio itself.

PID file location: `<workspace>/server.pid` (or similar).

- Source: https://github.com/fastapi/fastapi/discussions/6912 (HIGH)
- Source: https://fastapi.tiangolo.com/advanced/events/ (HIGH)

### F-03: Health Check / Readiness Probe

ML model servers need two distinct checks:

1. **Liveness**: Is the process alive? (PID exists and responds to signal 0, or TCP connect
   succeeds on port.)
2. **Readiness**: Is the model loaded and ready to serve inference? (HTTP GET to an endpoint
   returns 200 only after model weights are in memory.)

mlx-audio does not currently document a dedicated `/health` or `/ready` endpoint. Two
implementation options:

- **Option A (preferred)**: Poll `POST /v1/audio/speech` with a trivial input (single
  character) until it returns 200. This confirms both server and model are ready. First
  successful response means warm.
- **Option B**: Poll `GET /` or the web UI root for 200, then issue a single warm-up TTS
  request. Two-phase but slightly more robust if the model loads asynchronously.

The CLI `audio server start` command should block until readiness passes or a timeout (e.g.,
120s) expires, returning a structured error on timeout.

Best practice from ML serving (vLLM, TF Serving, Triton): separate liveness from readiness.
Liveness should be lightweight (TCP or /health). Readiness should verify model is loaded.

- Source: https://cloud.google.com/blog/products/containers-kubernetes/kubernetes-best-practices-setting-up-health-checks-with-readiness-and-liveness-probes (HIGH)
- Source: https://llm-d.ai/docs/usage/readiness-probes (MEDIUM)
- Source: https://www.tekhnoal.com/health-checks-for-ml-model-deployments (MEDIUM)

### F-04: Model Warm-Up

Qwen3-TTS model loading on Apple Silicon takes 10--30 seconds (PROJECT.md estimate). mlx-audio
optimized models use ~2--3 GB RAM. After loading, first inference may be slower due to Metal
shader compilation.

Strategy: After health check passes, issue one warm-up TTS request with a short Chinese
sentence (e.g., single character). Discard the output. This ensures Metal shaders are compiled
and caches are hot before the agent makes real requests.

The warm-up request should be part of `audio server start` -- the command only returns success
after warm-up completes.

- Source: https://github.com/Blaizzy/mlx-audio (MEDIUM)
- Source: PROJECT.md context section (HIGH)

### F-05: Chinese Mandarin Voice Selection

Qwen3-TTS CustomVoice models provide 9 preset speakers. The Chinese-native voices are:

| Speaker    | Description                                            | Language          |
|------------|--------------------------------------------------------|-------------------|
| Vivian     | Bright, slightly edgy young female voice               | Chinese           |
| Serena     | Warm, gentle young female voice                        | Chinese           |
| Uncle_Fu   | Seasoned male voice with a low, mellow timbre          | Chinese           |
| Dylan      | Youthful Beijing male voice, clear natural timbre      | Chinese (Beijing) |
| Eric       | Lively Chengdu male voice, slightly husky brightness   | Chinese (Sichuan) |

English-native speakers (Ryan, Aiden) and others (Ono_Anna/Japanese, Sohee/Korean) can also
speak Chinese but with non-native characteristics.

For language learning, Vivian or Serena are the best defaults -- clear female Mandarin voices
without regional dialect characteristics. Uncle_Fu provides a male alternative.

Each speaker has a native language but can speak any of the 10 supported languages. The
`language` parameter can be set to "Chinese" explicitly or "Auto" for detection.

- Source: https://huggingface.co/Qwen/Qwen3-TTS-12Hz-1.7B-CustomVoice (HIGH)
- Source: https://github.com/QwenLM/Qwen3-TTS (HIGH)

### F-06: Audio Output Format

mlx-audio supports WAV natively (no dependencies). MP3 and FLAC require ffmpeg.

The existing xuezh codebase already has ffmpeg as a dependency and conversion logic in
`buildConvertCommand()`. Supported output formats: wav, ogg, mp3.

The OpenAI API contract supports: mp3, opus, aac, flac, wav, pcm.

For the language learning use case:
- **WAV** for maximum compatibility and zero conversion latency
- **OGG/Opus** for space-efficient storage (existing default in xuezh)
- Convert server WAV output to desired format via existing ffmpeg pipeline

Qwen3-TTS outputs at 24000 Hz sample rate. The existing xuezh WAV conversion targets 16000 Hz
mono (adequate for speech).

- Source: https://pypi.org/project/mlx-audio/ (HIGH)
- Source: https://platform.openai.com/docs/api-reference/audio/createSpeech (HIGH)
- Source: existing codebase audio.go (HIGH)

### F-07: Error Handling

Critical error scenarios for the TTS server:

1. **Server not running**: HTTP connection refused. CLI should return structured error with
   hint to run `audio server start`.
2. **Server crashed**: PID file exists but process dead. CLI should detect stale PID, clean up,
   return error.
3. **Model OOM**: Possible on machines with insufficient unified memory. Server process dies.
   Same as crash handling.
4. **Text too long**: OpenAI API has 4096 char limit. Qwen3-TTS handles variable lengths but
   longer text increases latency. For language learning, typical inputs are short (single
   sentences, vocabulary words). No chunking needed for v1.
5. **Timeout**: TTS request taking too long. Go HTTP client should have configurable timeout
   (default 30s, generous for short text).
6. **Port conflict**: Desired port already in use. Detect on start, return structured error.

All errors must be returned as structured JSON in the existing envelope format. No panics, no
unstructured stderr.

- Source: https://community.openai.com/t/why-does-tts-work-on-small-text-but-gives-out-error-on-larger-text/498300 (MEDIUM)
- Source: existing codebase error handling patterns (HIGH)

### F-08: Edge-TTS Fallback

Keep edge-tts as a fallback backend. When the local server is not running (or fails), the
agent can specify `--backend edge-tts` to use the internet-dependent path. This is the existing
behavior -- no code changes needed, just preserve it.

The backend selection should be explicit per-call, not automatic failover. The agent decides
which backend to use based on server status.

- Source: PROJECT.md requirements (HIGH)

---

## Differentiators (Nice to Have)

### D-01: Emotion/Style Control via Instruct Parameter

Qwen3-TTS CustomVoice supports an `instruct` parameter for natural language emotion control.
Examples relevant to language learning:

- "Speak slowly and clearly" -- for beginner learners
- "Speak at natural conversational speed" -- for intermediate
- "Speak with enthusiasm" -- for encouragement
- Tone instructions in Chinese: "用特别愤怒的语气说" (speak in angry tone)

The 1.7B model has stronger instruction-following than 0.6B.

This could be exposed as an optional `--instruct` flag on the TTS command. The agent could
use it to vary pronunciation demonstrations (slow/clear for new words, natural speed for
review).

Implementation: Pass through to the `voice` or `instructions` field in the OpenAI-compatible
API call. Need to verify how mlx-audio maps this to Qwen3-TTS's instruct parameter.

- Source: https://huggingface.co/Qwen/Qwen3-TTS-12Hz-1.7B-CustomVoice (HIGH)
- Source: https://github.com/QwenLM/Qwen3-TTS (HIGH)

### D-02: Dialect Voices for Advanced Learners

Qwen3-TTS supports Chinese dialect speakers:
- Dylan: Beijing dialect
- Eric: Sichuan dialect

For advanced learners studying regional pronunciation differences, the agent could explicitly
select dialect voices. Low priority for v1 but zero additional implementation cost -- just
use a different voice name.

- Source: https://github.com/QwenLM/Qwen3-TTS (HIGH)

### D-03: Streaming Audio Output

Qwen3-TTS supports streaming with ~97ms first-packet latency (0.6B) and ~101ms (1.7B). Audio
packets are ~320ms each.

mlx-audio provides streaming via PCM samples with timing metrics.

For language learning, streaming is less critical than for conversational AI -- the learner
is waiting for a complete audio clip to play, not having a real-time conversation. However,
streaming could reduce perceived latency for longer sentences.

The OpenAI API supports `stream_format` parameter (values: "audio" or "sse").

Implementation complexity is moderate: requires chunked HTTP response handling in Go,
progressive audio assembly, and a player that can start before download completes.

- Source: https://github.com/QwenLM/Qwen3-TTS (HIGH)
- Source: https://github.com/rekuenkdr/Qwen3-TTS-streaming (MEDIUM)
- Source: https://huggingface.co/Qwen/Qwen3-TTS-Tokenizer-12Hz (HIGH)

### D-04: Voice Design (Create Custom Voices)

Qwen3-TTS VoiceDesign model can create entirely new voices from text descriptions:
"A middle-aged female professor with a slight Beijing accent speaking slowly."

Workflow: (1) VoiceDesign generates a reference clip, (2) clone that voice for reuse.

For language learning, this could create a persistent "teacher" voice. However, this requires
the VoiceDesign model variant (separate from CustomVoice), adds complexity, and the 9 preset
voices are likely sufficient.

- Source: https://huggingface.co/Qwen/Qwen3-TTS-12Hz-1.7B-VoiceDesign (HIGH)
- Source: https://github.com/QwenLM/Qwen3-TTS (HIGH)

### D-05: Server Auto-Start on First TTS Request

Instead of requiring explicit `audio server start`, the CLI could auto-start the server on
the first TTS request with `--backend local` and keep it running. Agent would not need to
manage lifecycle explicitly.

Risk: Cold start adds 10--30s to first request. Agent may not understand why first call is
slow. Explicit lifecycle is more predictable and matches the agent flow described in PROJECT.md.

- Source: architectural consideration (MEDIUM)

### D-06: Multiple Model Support / Model Selection

mlx-audio supports multiple TTS models (Kokoro, Qwen3-TTS, Chatterbox, Dia, OuteTTS, Spark,
Soprano). The server could potentially be started with different models via the `model`
parameter in the API request.

For v1, hardcoding to Qwen3-TTS CustomVoice is correct -- it has the best Chinese quality.
Future versions could expose model selection.

- Source: https://github.com/Blaizzy/mlx-audio (HIGH)

---

## Anti-Features (Commonly Requested but Problematic)

### A-01: Voice Cloning

Qwen3-TTS Base model supports voice cloning from 3 seconds of reference audio. While
technically impressive, this is explicitly out of scope per PROJECT.md: "Custom voice training
or voice cloning -- use stock voices." Voice cloning adds:

- Privacy/consent concerns with reference audio
- Storage and management of voice profiles
- Quality variability with different reference samples
- Additional model variant needed (Base instead of CustomVoice)

The 9 preset voices provide sufficient variety for language learning.

- Source: PROJECT.md out-of-scope section (HIGH)

### A-02: Automatic Backend Failover

Tempting to auto-fallback from local to edge-tts when the server is down. Problems:

- Masks server failures (agent never learns the server crashed)
- Different voice characteristics between backends create inconsistent learning experience
- Latency profiles differ dramatically (local ~100ms vs edge-tts ~1-3s)
- Edge-tts requires internet; "fallback" may also fail

Better: Return structured error when local backend fails. Agent decides retry strategy.

- Source: architectural consideration (MEDIUM)

### A-03: Running Multiple TTS Servers

Running multiple mlx-audio instances (different models, different ports). Problems:

- Each instance uses 2--3 GB RAM on Mac Mini M4 (which has limited unified memory)
- Port management complexity
- No demonstrated need -- one model serves all language learning use cases

- Source: PROJECT.md constraints (HIGH)

### A-04: HTTPS/TLS for Localhost

The server runs on localhost only. Adding TLS adds:

- Certificate management complexity
- No security benefit (traffic never leaves the machine)
- Breaks simple curl debugging

HTTP on 127.0.0.1 is the correct choice.

- Source: architectural consideration (HIGH)

### A-05: Web UI Exposure

mlx-audio includes a web UI for interactive use. For this project, the UI is unnecessary:

- The consumer is a CLI/agent, not a human
- Exposing a UI creates an unnecessary attack surface
- The server should be headless, API-only

- Source: architectural consideration (HIGH)

### A-06: Real-Time Conversation Mode

Qwen3-TTS streaming + low latency enables conversational AI scenarios. For language learning
with SRS, the interaction pattern is discrete (generate clip, play, assess) not continuous.
Building a real-time conversation pipeline adds massive complexity for no learning benefit
in the SRS context.

- Source: architectural consideration (MEDIUM)

---

## Feature Dependencies

```
F-02 (Server Lifecycle) ──> F-03 (Health Check) ──> F-04 (Model Warm-Up)
                                                         │
F-01 (TTS Endpoint) ──> F-05 (Voice Selection)          │
        │                                                 │
        └──> F-06 (Audio Format) ──> F-07 (Error Handling)
                                          │
F-08 (Edge-TTS Fallback) ────────────────┘

D-01 (Emotion Control) depends on F-01, F-05
D-02 (Dialect Voices) depends on F-05
D-03 (Streaming) depends on F-01
D-04 (Voice Design) depends on F-01, separate model variant
D-05 (Auto-Start) depends on F-02, F-03
```

Critical path: F-02 -> F-03 -> F-04 -> F-01 -> F-07

The server must be startable and health-checked before any TTS calls can be made.
Voice selection and audio format are configuration concerns that parallelize with
the server lifecycle work.

---

## MVP Definition

The MVP enables this agent flow:

```
Agent: "Start a Chinese lesson"
  CLI: audio server start --port 8787 --model mlx-community/Qwen3-TTS-12Hz-1.7B-CustomVoice-bf16
  CLI: (spawns mlx_audio.server, waits for readiness, runs warm-up) -> {"status": "ready"}

Agent: "Pronounce 你好"
  CLI: audio tts --text "你好" --voice Vivian --backend local --output lesson.ogg
  CLI: (POST /v1/audio/speech, convert WAV->OGG) -> {"out": "artifacts/.../lesson.ogg"}

Agent: "End lesson"
  CLI: audio server stop
  CLI: (SIGTERM to PID, wait, cleanup) -> {"status": "stopped"}
```

MVP features (all Table Stakes):
- F-02: Server start/stop/status with PID tracking
- F-03: Health check polling during startup
- F-04: Model warm-up on start
- F-01: TTS via HTTP POST to /v1/audio/speech
- F-05: Voice selection (default: Vivian for Chinese)
- F-06: WAV output with conversion to ogg/mp3 via existing ffmpeg pipeline
- F-07: Structured error handling for all failure modes
- F-08: Edge-tts fallback preserved (no changes)

MVP excludes: Streaming (D-03), Emotion control (D-01), Voice design (D-04),
Auto-start (D-05).

---

## Feature Prioritization Matrix

| Feature | Impact | Effort | Risk | Priority |
|---------|--------|--------|------|----------|
| F-02 Server Lifecycle      | Critical | Medium | Low    | P0 -- blocks everything    |
| F-03 Health Check           | Critical | Low    | Low    | P0 -- part of F-02         |
| F-04 Model Warm-Up          | High     | Low    | Low    | P0 -- part of F-02         |
| F-01 TTS Endpoint           | Critical | Low    | Low    | P0 -- core functionality   |
| F-05 Voice Selection        | High     | Low    | Low    | P0 -- config, not code     |
| F-06 Audio Format           | High     | None   | None   | P0 -- existing code works  |
| F-07 Error Handling         | High     | Medium | Low    | P0 -- must be robust       |
| F-08 Edge-TTS Fallback      | Medium   | None   | None   | P0 -- already exists       |
| D-01 Emotion Control        | Medium   | Low    | Medium | P1 -- low effort, unclear mapping |
| D-02 Dialect Voices         | Low      | None   | None   | P1 -- just a voice name    |
| D-03 Streaming              | Low      | High   | Medium | P2 -- short text, low benefit |
| D-05 Auto-Start             | Low      | Medium | Medium | P2 -- explicit is better   |
| D-04 Voice Design           | Low      | High   | High   | P3 -- separate model, complexity |
| D-06 Model Selection        | Low      | Low    | Low    | P3 -- no need yet          |

---

## Technical Specifications Summary

### mlx-audio Server

- Start command: `mlx_audio.server --host 127.0.0.1 --port <port>`
- Default port: 8000
- Framework: FastAPI/Uvicorn
- TTS endpoint: `POST /v1/audio/speech` (OpenAI-compatible)
- Request body: `{"model": "<model-path>", "input": "<text>", "voice": "<voice>"}`
- Response: audio bytes (WAV)
- Audio formats: WAV (native), MP3/FLAC (requires ffmpeg)
- Graceful shutdown: SIGTERM handled by Uvicorn natively

### Qwen3-TTS Model

- Recommended variant: `mlx-community/Qwen3-TTS-12Hz-1.7B-CustomVoice-bf16`
- RAM usage: ~2--3 GB on Apple Silicon
- Sample rate: 24000 Hz output
- Languages: Chinese, English, Japanese, Korean, German, French, Russian, Portuguese, Spanish, Italian
- Chinese dialects: Mandarin, Beijing, Sichuan (via speaker selection)
- Speakers: 9 preset (5 Chinese-native, 2 English-native, 1 Japanese, 1 Korean)
- Instruct control: natural language emotion/style instructions (CustomVoice only)
- Streaming latency: ~97ms first packet (0.6B), ~101ms (1.7B)
- License: Apache 2.0

### OpenAI TTS API Contract (reference)

- Endpoint: `POST /v1/audio/speech`
- Required: `model`, `input`, `voice`
- Optional: `response_format` (mp3|opus|aac|flac|wav|pcm, default mp3), `speed` (0.25--4.0, default 1.0), `instructions`, `stream_format` (audio|sse)
- Response: audio file content or stream

---

*Research completed 2026-02-28. Sources verified against latest available documentation.*
