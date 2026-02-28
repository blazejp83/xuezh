# Technology Stack Research: mlx-audio + Go CLI on Apple Silicon

> Researched: 2026-02-28
> Target hardware: Mac Mini M4
> Purpose: Local TTS backend for xuezh language-learning CLI

---

## Core Technologies

| Component       | Package / Tool                    | Version         | Role                                    | Confidence |
|-----------------|-----------------------------------|-----------------|-----------------------------------------|------------|
| TTS Engine      | `mlx-audio`                       | 0.3.1           | Local TTS via MLX on Apple Silicon      | HIGH       |
| ML Framework    | `mlx`                             | >= 0.25.2       | Apple Silicon ML array framework        | HIGH       |
| Python          | CPython (arm64 native)            | >= 3.10         | Runtime for mlx-audio                   | HIGH       |
| Go              | Go                                | 1.22            | CLI application (xuezh)                 | HIGH       |
| Model Hub       | `huggingface_hub`                 | >= 0.27.0       | Model download and caching              | HIGH       |
| Audio Encoding  | FFmpeg                            | any recent      | MP3/FLAC encoding (WAV works without)   | HIGH       |

Sources:
- [mlx-audio PyPI](https://pypi.org/project/mlx-audio/) - version 0.3.1, released 2026-01-29
- [MLX Install Docs](https://ml-explore.github.io/mlx/build/html/install.html) - macOS >= 14.0, Python >= 3.10
- [mlx-audio GitHub](https://github.com/Blaizzy/mlx-audio)
- [DeepWiki - Installation & Requirements](https://deepwiki.com/Blaizzy/mlx-audio/1.1-installation-and-requirements)

---

## System Requirements

| Requirement       | Minimum                              | Recommended                   | Notes                                  |
|-------------------|--------------------------------------|-------------------------------|----------------------------------------|
| macOS version     | 14.0 (Sonoma)                        | 15.x (Sequoia)                | MLX requires macOS >= 14.0             |
| Apple Silicon     | M1                                   | M4                            | All M-series supported; no Intel       |
| Python            | 3.10 (native arm64)                  | 3.12                          | Must be arm64, NOT Rosetta x86_64      |
| Unified Memory    | 8 GB                                 | 16 GB                         | 0.6B-4bit needs ~1.7 GB on disk       |
| Disk (model)      | ~1.7 GB (0.6B-4bit)                  | ~2 GB (0.6B-8bit)             | Cached in HuggingFace hub dir          |
| FFmpeg            | Optional                             | Installed via `brew`           | Only needed for MP3/FLAC output        |

Verification: run `python -c "import platform; print(platform.processor())"` -- must print `arm`, not `i386`.

Source: [MLX Install Docs](https://ml-explore.github.io/mlx/build/html/install.html) - HIGH confidence

---

## mlx-audio Dependencies (pinned in 0.3.1)

### Core (always installed)

| Package          | Version Constraint | Purpose                             |
|------------------|--------------------|-------------------------------------|
| `mlx`            | >= 0.25.2          | Apple Silicon ML framework          |
| `mlx-lm`         | == 0.30.5          | Language model utilities for MLX    |
| `transformers`   | == 5.0.0rc3        | HuggingFace tokenizers / model loading |
| `numpy`          | >= 1.26.4          | Array operations                    |
| `numba`          | >= 0.60.0          | JIT compilation for audio processing |
| `librosa`        | == 0.11.0          | Audio feature extraction            |
| `sounddevice`    | == 0.5.3           | Audio playback                      |
| `miniaudio`      | >= 1.61            | Audio I/O                           |
| `pyloudnorm`     | >= 0.2.0           | Loudness normalization              |
| `huggingface_hub`| >= 0.27.0          | Model download/caching              |
| `tqdm`           | >= 4.67.1          | Progress bars                       |

### Optional Extra Groups

| Extra       | Install Command                       | Adds                                    |
|-------------|---------------------------------------|-----------------------------------------|
| `[tts]`     | `pip install mlx-audio[tts]`          | misaki, phonemizer, spacy               |
| `[stt]`     | `pip install mlx-audio[stt]`          | tiktoken, mistral-common                |
| `[server]`  | `pip install mlx-audio[server]`       | fastapi, uvicorn                        |
| `[all]`     | `pip install mlx-audio[all]`          | Everything above                        |
| `[dev]`     | `pip install -e ".[dev]"`             | Dev/test tools                          |

Source: [DeepWiki - Installation & Requirements](https://deepwiki.com/Blaizzy/mlx-audio/1.1-installation-and-requirements) - HIGH confidence

---

## Installation Commands

### Quick Setup (recommended for xuezh)

```bash
# 1. Create isolated Python environment
python3 -m venv ~/.local/share/xuezh/mlx-env
source ~/.local/share/xuezh/mlx-env/bin/activate

# 2. Install mlx-audio with server + TTS extras
pip install "mlx-audio[server,tts]"

# 3. Optional: FFmpeg for MP3/FLAC output
brew install ffmpeg
```

### Alternative: uv (faster, no venv management)

```bash
uv tool install --force mlx-audio --prerelease=allow
# or from git for bleeding edge:
uv tool install --force git+https://github.com/Blaizzy/mlx-audio.git --prerelease=allow
```

### Verify Installation

```bash
python -c "import mlx_audio; print('mlx-audio OK')"
python -c "import mlx; print('MLX OK')"
```

Source: [mlx-audio README](https://github.com/Blaizzy/mlx-audio/blob/main/README.md) - HIGH confidence

---

## Qwen3-TTS Model Identifiers (mlx-community on HuggingFace)

### Recommended for xuezh: `mlx-community/Qwen3-TTS-12Hz-0.6B-Base-8bit`

Rationale: Best balance of quality and size. 17.7k downloads (most popular). 1.99 GB on disk.

### Full Model Catalog

#### Base Models (predefined voices, `generate()` method)

| Model ID                                              | Params | Disk Size | Quant | Downloads |
|-------------------------------------------------------|--------|-----------|-------|-----------|
| `mlx-community/Qwen3-TTS-12Hz-0.6B-Base-bf16`        | 0.9B   | ~2.5 GB   | bf16  | 1,860     |
| `mlx-community/Qwen3-TTS-12Hz-0.6B-Base-8bit`        | 0.5B   | 1.99 GB   | 8-bit | 17,700    |
| `mlx-community/Qwen3-TTS-12Hz-0.6B-Base-6bit`        | 0.5B   | ~1.8 GB   | 6-bit | 108       |
| `mlx-community/Qwen3-TTS-12Hz-0.6B-Base-5bit`        | 0.5B   | ~1.6 GB   | 5-bit | 66        |
| `mlx-community/Qwen3-TTS-12Hz-0.6B-Base-4bit`        | 0.4B   | 1.71 GB   | 4-bit | 1,303     |

#### CustomVoice Models (emotion control, `generate_custom_voice()` method)

| Model ID                                                    | Params | Quant | Downloads |
|-------------------------------------------------------------|--------|-------|-----------|
| `mlx-community/Qwen3-TTS-12Hz-0.6B-CustomVoice-bf16`       | 0.9B   | bf16  | 445       |
| `mlx-community/Qwen3-TTS-12Hz-0.6B-CustomVoice-8bit`       | 0.5B   | 8-bit | 14,100    |
| `mlx-community/Qwen3-TTS-12Hz-0.6B-CustomVoice-6bit`       | 0.5B   | 6-bit | 57        |
| `mlx-community/Qwen3-TTS-12Hz-0.6B-CustomVoice-4bit`       | 0.4B   | 4-bit | 345       |

#### VoiceDesign Models (create arbitrary voices, `generate_voice_design()` method)

| Model ID                                                    | Params | Quant | Downloads |
|-------------------------------------------------------------|--------|-------|-----------|
| `mlx-community/Qwen3-TTS-12Hz-1.7B-VoiceDesign-bf16`       | 2B     | bf16  | 2,490     |
| `mlx-community/Qwen3-TTS-12Hz-1.7B-VoiceDesign-8bit`       | 0.8B   | 8-bit | 2,330     |
| `mlx-community/Qwen3-TTS-12Hz-1.7B-VoiceDesign-6bit`       | 0.7B   | 6-bit | 89        |
| `mlx-community/Qwen3-TTS-12Hz-1.7B-VoiceDesign-4bit`       | 0.6B   | 4-bit | 344       |

### Available Voices (Base models)

| Voice    | Language | Notes              |
|----------|----------|--------------------|
| Chelsie  | English  | Default English    |
| Ryan     | English  |                    |
| Aiden    | English  |                    |
| Vivian   | Chinese  |                    |
| Serena   | Chinese  |                    |
| Uncle_Fu | Chinese  |                    |
| Dylan    | Chinese  | Beijing dialect    |
| Eric     | Chinese  | Sichuan dialect    |

### Supported Languages

Qwen3-TTS supports: Chinese (ZH), English (EN), Japanese (JA), Korean (KO), and more.

Sources:
- [HuggingFace Collection: Qwen3-TTS](https://huggingface.co/collections/mlx-community/qwen3-tts) - HIGH confidence
- [Qwen3-TTS-12Hz-0.6B-Base-4bit](https://huggingface.co/mlx-community/Qwen3-TTS-12Hz-0.6B-Base-4bit) - HIGH confidence
- [Qwen3-TTS-12Hz-0.6B-Base-8bit](https://huggingface.co/mlx-community/Qwen3-TTS-12Hz-0.6B-Base-8bit) - HIGH confidence
- [mlx-audio Qwen3-TTS README](https://github.com/Blaizzy/mlx-audio/blob/main/mlx_audio/tts/models/qwen3_tts/README.md) - HIGH confidence

---

## mlx-audio Server: OpenAI-Compatible API

### Starting the Server

```bash
# Basic
mlx_audio.server

# With custom host/port
mlx_audio.server --host 0.0.0.0 --port 8000

# With verbose logging
mlx_audio.server --verbose
```

Server is built on FastAPI + Uvicorn. Requires `[server]` extra (`pip install mlx-audio[server]`).

### API Endpoints

| Method   | Endpoint                    | Purpose                              |
|----------|-----------------------------|--------------------------------------|
| `POST`   | `/v1/audio/speech`          | Generate speech from text (TTS)      |
| `POST`   | `/v1/audio/transcriptions`  | Transcribe audio file (STT)          |
| `GET`    | `/v1/models`                | List loaded models                   |
| `POST`   | `/v1/models`                | Load a model by name                 |
| `DELETE` | `/v1/models`                | Unload a model                       |

### POST /v1/audio/speech (primary endpoint for xuezh)

**Request Body (JSON):**

```json
{
  "model": "mlx-community/Qwen3-TTS-12Hz-0.6B-Base-8bit",
  "input": "Hello, this is a test.",
  "voice": "Chelsie",
  "speed": 1.0
}
```

| Parameter  | Type   | Required | Description                              |
|------------|--------|----------|------------------------------------------|
| `model`    | string | yes      | HuggingFace model ID or local path       |
| `input`    | string | yes      | Text to convert to speech                |
| `voice`    | string | no       | Voice preset name (e.g., "Chelsie")      |
| `speed`    | float  | no       | Speech speed multiplier (default 1.0)    |

**Response:** Audio data in WAV format (binary).

**Example curl:**

```bash
curl -X POST http://localhost:8000/v1/audio/speech \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mlx-community/Qwen3-TTS-12Hz-0.6B-Base-8bit",
    "input": "你好世界",
    "voice": "Vivian"
  }' \
  --output output.wav
```

### Go Integration Pattern

The Go CLI should:
1. Spawn the mlx-audio server as a child process
2. Wait for the server to be ready (poll `/v1/models` or health endpoint)
3. Make HTTP POST requests to `/v1/audio/speech`
4. Receive WAV audio bytes in the response body
5. Optionally convert with ffmpeg to ogg/mp3

```go
// Conceptual Go integration
cmd := exec.Command("python", "-m", "mlx_audio.server", "--host", "127.0.0.1", "--port", "8321")
// POST to http://127.0.0.1:8321/v1/audio/speech with JSON body
// Response body is WAV audio bytes
```

Sources:
- [mlx-audio GitHub README](https://github.com/Blaizzy/mlx-audio) - HIGH confidence
- [mlx-audio PyPI](https://pypi.org/project/mlx-audio/0.2.7/) - MEDIUM confidence (endpoint details from older version docs)

---

## CLI Commands (non-server usage)

### TTS Generation

```bash
# Basic generation
mlx_audio.tts.generate \
  --model mlx-community/Qwen3-TTS-12Hz-0.6B-Base-8bit \
  --text "你好世界"

# With voice, speed, and playback
mlx_audio.tts.generate \
  --model mlx-community/Qwen3-TTS-12Hz-0.6B-Base-8bit \
  --text "Hello!" \
  --voice Chelsie \
  --speed 1.2 \
  --play

# Output to specific path
mlx_audio.tts.generate \
  --model mlx-community/Qwen3-TTS-12Hz-0.6B-Base-8bit \
  --text "Hello!" \
  --output_path ./my_audio

# Generation parameters
mlx_audio.tts.generate \
  --model mlx-community/Qwen3-TTS-12Hz-0.6B-Base-8bit \
  --text "Hello!" \
  --temperature 0.4 \
  --top_p 0.9 \
  --top_k 50 \
  --play
```

### Python API (for reference)

```python
from mlx_audio.tts.utils import load_model

model = load_model("mlx-community/Qwen3-TTS-12Hz-0.6B-Base-8bit")
results = list(model.generate(
    text="你好，欢迎使用MLX Audio！",
    voice="Vivian",
    language="Chinese",
))
audio = results[0].audio  # mx.array
```

Source: [mlx-audio README](https://github.com/Blaizzy/mlx-audio/blob/main/README.md) - HIGH confidence

---

## Performance Estimates

### Qwen3-TTS 0.6B on Apple Silicon (via MLX)

| Metric                    | Value              | Notes                                    | Confidence |
|---------------------------|--------------------|------------------------------------------|------------|
| RTF (Real-Time Factor)    | ~1.8-2.2x          | Comparable to RTX 3060 performance       | MEDIUM     |
| First token latency       | ~50-80ms (est.)    | Extrapolated from GPU benchmarks         | LOW        |
| Model load time           | 2-5 seconds (est.) | First load; subsequent from HF cache     | LOW        |
| Memory (0.6B-4bit)        | ~2-3 GB unified    | Includes model + inference overhead      | MEDIUM     |
| Memory (0.6B-8bit)        | ~3-4 GB unified    | Includes model + inference overhead      | MEDIUM     |
| Disk (0.6B-4bit)          | 1.71 GB            | Measured from HuggingFace                | HIGH       |
| Disk (0.6B-8bit)          | 1.99 GB            | Measured from HuggingFace                | HIGH       |

### GPU Benchmarks (for reference, not Apple Silicon)

| Hardware  | RTF (0.6B) | VRAM    | First Token | Source                |
|-----------|------------|---------|-------------|-----------------------|
| RTX 4090  | 0.38       | 2.9 GB  | 52ms        | qwen3-tts.app         |
| RTX 3090  | 0.52       | 3.1 GB  | 78ms        | qwen3-tts.app         |
| M-series  | ~1.8-2.2   | unified | ~50-80ms    | Extrapolated          |

RTF < 1.0 means faster than real-time. RTF ~2.0 on M-series means generating 1 second of audio takes ~2 seconds. For short TTS snippets in a language learning app, this is acceptable.

Sources:
- [Qwen3-TTS Performance Benchmarks](https://qwen3-tts.app/blog/qwen3-tts-performance-benchmarks-hardware-guide-2026) - MEDIUM confidence
- [qwen3-tts-apple-silicon GitHub](https://github.com/kapi2800/qwen3-tts-apple-silicon) - MEDIUM confidence

---

## Alternatives Considered

### edge-tts (current backend in xuezh)

| Aspect            | Details                                                  |
|-------------------|----------------------------------------------------------|
| Package           | `pip install edge-tts`                                   |
| Type              | Cloud-based (Microsoft Edge online TTS service)          |
| Requires internet | **YES** -- fundamental limitation, cannot work offline   |
| API key needed    | No                                                       |
| Quality           | Good (Microsoft Neural voices)                           |
| Latency           | ~200-500ms (network-dependent)                           |
| Chinese voices    | `zh-CN-XiaoxiaoNeural`, `zh-CN-YunxiNeural`, etc.       |
| Risk              | Microsoft could rate-limit or discontinue at any time    |
| License           | Uses undocumented Microsoft API                          |

**Verdict:** Good for quick prototyping, but unsuitable for reliable offline language learning. Currently used by xuezh (`buildTTSCommand` in `audio.go`).

Source: [edge-tts GitHub](https://github.com/rany2/edge-tts) - HIGH confidence

### CosyVoice (Alibaba)

| Aspect            | Details                                                  |
|-------------------|----------------------------------------------------------|
| Package           | `pip install cosyvoice` (complex setup)                  |
| Type              | Local inference (PyTorch-based)                          |
| Apple Silicon     | Via MPS backend; not MLX-native, suboptimal performance  |
| Quality           | Excellent voice cloning                                  |
| Complexity        | Heavy dependencies, complex installation                 |
| Streaming         | CosyVoice 2.0 supports unified streaming                |

**Verdict:** Strong model but not optimized for Apple Silicon. No MLX port. Heavier than mlx-audio.

### Piper TTS

| Aspect            | Details                                                  |
|-------------------|----------------------------------------------------------|
| Package           | Binary distribution or `pip install piper-tts`           |
| Type              | Local inference (ONNX-based)                             |
| Apple Silicon     | Runs but not GPU-accelerated (CPU only)                  |
| Quality           | Good for English; limited Chinese voice options           |
| Speed             | Very fast on CPU (<1s for short text)                    |
| Chinese support   | Limited / community voices only                          |

**Verdict:** Fast and lightweight but weak Chinese language support. No Apple Silicon GPU acceleration.

### Kokoro (via mlx-audio)

| Aspect            | Details                                                  |
|-------------------|----------------------------------------------------------|
| Model             | `mlx-community/Kokoro-82M-bf16`                         |
| Type              | Local via mlx-audio (same server infrastructure)         |
| Size              | 82M params -- very small                                 |
| Quality           | Good for its size                                        |
| Chinese support   | Yes (Mandarin via `lang_code="z"`)                       |
| Caveat            | Requires `pip install misaki[zh]` for Chinese            |

**Verdict:** Worth considering as a faster/lighter alternative within the same mlx-audio ecosystem. Could serve as a fallback model.

---

## What NOT to Use

| Technology              | Reason                                                           |
|-------------------------|------------------------------------------------------------------|
| PyTorch + MPS backend   | Suboptimal on Apple Silicon vs native MLX; higher memory usage   |
| OpenAI TTS API          | Cloud-based, costs money, requires API key, network dependency   |
| Google Cloud TTS        | Cloud-based, costs money, requires credentials                   |
| espeak-ng               | Very low quality, robotic voice                                  |
| macOS `say` command     | Low quality, no Chinese support worth using                      |
| Bark (suno-ai)          | Very slow, high memory usage, not optimized for Apple Silicon    |
| XTTS v2 (Coqui)        | Project discontinued; dependencies rot                           |
| Intel-based Python      | MLX will not work under Rosetta; must be native arm64 Python     |

---

## Version Compatibility Matrix

| mlx-audio | MLX      | Python  | transformers  | macOS   | Notes                        |
|-----------|----------|---------|---------------|---------|------------------------------|
| 0.3.1     | >= 0.25.2| >= 3.10 | == 5.0.0rc3   | >= 14.0 | Current release (2026-01-29) |
| 0.3.0     | >= 0.25.2| >= 3.10 | == 5.0.0rc3   | >= 14.0 | Model conversion version     |
| 0.2.x     | older    | >= 3.8  | varies        | >= 14.0 | Older; fewer models          |

**Key constraint:** `transformers == 5.0.0rc3` is pinned exactly. This may conflict with other packages in the same venv. Use an isolated environment.

**Key constraint:** `mlx-lm == 0.30.5` is pinned exactly. Same isolation recommendation.

Source: [DeepWiki - Installation & Requirements](https://deepwiki.com/Blaizzy/mlx-audio/1.1-installation-and-requirements) - HIGH confidence

---

## Architecture Decision: Go CLI + mlx-audio Server

### Chosen Pattern: Sidecar Process

```
xuezh (Go CLI)
  |
  |-- spawns --> mlx_audio.server (Python/FastAPI on localhost:8321)
  |                |
  |                |-- loads --> Qwen3-TTS model (MLX, Apple Silicon GPU)
  |
  |-- HTTP POST /v1/audio/speech --> receives WAV bytes
  |
  |-- (optional) ffmpeg --> convert to ogg/mp3
```

### Why This Pattern

1. **Language boundary:** Go cannot call MLX/Python directly; HTTP is the clean boundary
2. **OpenAI-compatible:** The `/v1/audio/speech` endpoint matches OpenAI's TTS API shape, making the Go client trivially simple
3. **Process isolation:** Python ML model crashes don't take down the Go CLI
4. **Reusable:** Same server can be used by other tools or a future web UI
5. **Existing pattern:** xuezh already shells out to `edge-tts` and `whisper` CLI tools

### Server Lifecycle (Go side)

1. Check if server already running (GET `/v1/models`)
2. If not, spawn `python -m mlx_audio.server --host 127.0.0.1 --port 8321`
3. Poll until server responds (health check loop, max ~30s for first model load)
4. Make TTS requests as needed
5. Optionally keep server alive between CLI invocations (daemon mode) or kill on exit

---

## Open Questions

1. **Streaming support:** Does `/v1/audio/speech` support chunked/streaming audio response for long text? (needs testing)
2. **Server health endpoint:** Is there a dedicated `/health` or `/ready` endpoint, or must we use `/v1/models`? (needs testing)
3. **Model preloading:** Can we tell the server to preload a specific model at startup via CLI flag? (not documented)
4. **Concurrent requests:** How does the server handle multiple simultaneous TTS requests? (single-threaded MLX inference likely serializes)
5. **WAV sample rate:** What sample rate does the Qwen3-TTS output? The Go side needs to know for ffmpeg conversion params.
6. **CustomVoice for language learning:** Should we use CustomVoice models for emotion/speed control in pronunciation practice?
