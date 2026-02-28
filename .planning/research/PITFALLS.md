# Pitfalls: Integrating mlx-audio (Qwen3-TTS) as a Subprocess Server from Go on macOS

Research date: 2026-02-28
Context: Go CLI managing a Python mlx-audio server running Qwen3-TTS (1.7B params, ~2-3GB RAM) on Mac Mini M4. Server started/stopped per learning session (~30-60 min).

---

## Critical Pitfalls (Top 6)

### 1. Zombie/Orphan Process Trees After Unclean Shutdown

**The Problem:** When Go launches `mlx_audio.server` via `exec.Command`, Python may spawn child processes (uvicorn workers, Metal shader compilation subprocesses). Sending SIGTERM or SIGKILL to the parent PID does not kill children. They become orphaned, reparented to PID 1, and hold the port and GPU memory indefinitely. Uvicorn 0.29+ has a known bug where child processes become orphaned on graceful shutdown.

**Prevention:**
- Always set `cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}` to create a new process group
- Kill the entire process group with `syscall.Kill(-pgid, syscall.SIGTERM)` (negative PID = process group)
- Implement a two-phase shutdown: SIGTERM first, wait 5s, then SIGKILL the process group
- On startup, scan for and kill stale processes on the target port before binding
- Store the PGID (not just PID) in state tracking

**Confidence:** HIGH -- well-documented across Go process management literature and uvicorn issue trackers.

**Sources:**
- [Killing a child process and all of its children in Go](https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773)
- [Killing process descendants in Go](https://sigmoid.at/post/2023/08/kill_process_descendants_golang/)
- [Uvicorn child processes not terminating](https://github.com/Kludex/uvicorn/issues/2289)
- [os/exec: Unable to kill a command on macOS](https://github.com/golang/go/issues/27440)

### 2. Memory Growth Leading to System OOM and Forced Reboot

**The Problem:** MLX has documented memory leaks during sustained inference sessions. The mlx_lm.server systematically goes out of memory during prolonged use. On macOS, the unified memory architecture means GPU memory pressure = system memory pressure. The jetsam kernel mechanism (macOS's OOM killer) will terminate the process, or in severe cases, force a system reboot. A 1.7B param model using ~2-3GB leaves limited headroom on systems with other apps running.

**Prevention:**
- Monitor memory via `os.exec` calls to `ps -o rss= -p <pid>` periodically
- Set a memory ceiling; restart the server if RSS exceeds threshold (e.g., 4GB)
- Consider periodic server restarts during long sessions (every 20-30 min)
- Use `mlx.core.metal.clear_cache()` if accessible via API, or accept that restart is the only reliable cache clear
- Warn the user if system memory pressure is high before starting the server

**Confidence:** HIGH -- memory leak in mlx_lm.server confirmed in GitHub issues; MLX/Metal memory growth reported independently.

**Sources:**
- [Memory leak in mlx_lm.server](https://github.com/ml-explore/mlx-examples/issues/1124)
- [Memory leak in MLX / Metal / MPS](https://github.com/ml-explore/mlx/issues/755)
- [macOS jetsam memory pressure handling](https://newosxbook.com/articles/MemoryPressure.html)

### 3. Cold Start: Model Loading + Metal Shader Compilation = 15-40s Wait

**The Problem:** First request after server start hits two sequential cold-start penalties: (a) model weight loading from disk into unified memory (~2-3GB), and (b) Metal shader JIT compilation for GPU kernels. The MLX_METAL_JIT flag causes run-time compilation on first use, adding hundreds of milliseconds to several seconds. Total cold start can be 15-40s. The Metal shader cache persists across reboots, so shader compilation is only slow on very first use, but model loading happens every server start.

**Prevention:**
- Implement a readiness probe: poll a lightweight endpoint (e.g., `GET /v1/models`) before declaring the server ready
- Send a "warmup" TTS request with a short dummy text immediately after readiness to trigger lazy initialization and shader compilation
- Show a progress indicator to the user during startup ("Loading model... Warming up...")
- Do NOT set a short timeout on the readiness check; allow up to 60s
- Pre-download models during install rather than at first server start

**Confidence:** HIGH -- MLX documentation explicitly describes JIT cold-start costs; model loading time is inherent to size.

**Sources:**
- [MLX Metal JIT documentation](https://ml-explore.github.io/mlx/build/html/install.html)
- [Production-Grade Local LLM Inference on Apple (31s init time)](https://arxiv.org/pdf/2511.05502)
- [MLX unified memory documentation](https://ml-explore.github.io/mlx/build/html/usage/unified_memory.html)

### 4. Port Conflicts: AirPlay, TIME_WAIT, and Firewall Prompts

**The Problem:** Three distinct port issues on macOS:
1. **AirPlay Receiver** occupies port 5000 (and 7000) by default on macOS Monterey+. If mlx-audio defaults to port 5000, it silently fails or gets a 403 from AirPlay's daemon.
2. **TIME_WAIT after crash:** If the server crashes or is killed, the TCP port enters TIME_WAIT for 1-4 minutes. Immediate restart fails with EADDRINUSE.
3. **Firewall prompt:** Binding to `0.0.0.0` triggers macOS firewall dialog "Do you want the application to accept incoming connections?" every launch. Binding to `127.0.0.1` does not.

**Prevention:**
- Use a non-standard port (e.g., 8778) and always bind to `127.0.0.1`, never `0.0.0.0`
- Before starting, check if the port is in use: `lsof -ti:<port>` and kill any stale process
- Ensure Python server sets `SO_REUSEADDR` (uvicorn does this by default)
- If port is occupied, fall back to an alternative port and communicate it
- Document that AirPlay Receiver should be disabled or a non-5000 port used

**Confidence:** HIGH -- AirPlay port 5000 conflict is the most common Flask/Python server issue on macOS since 2021.

**Sources:**
- [macOS port 5000 AirPlay conflict](https://github.com/dgtlmoon/changedetection.io/issues/3401)
- [Port 5000 used by Control Center in macOS Ventura](https://nono.ma/port-5000-used-by-control-center-in-macos-ventura-controlce)
- [SO_REUSEADDR and TIME_WAIT](https://hea-www.harvard.edu/~fine/Tech/addrinuse.html)

### 5. Go os/exec Pipe Deadlock and Wait() Hang

**The Problem:** When Go captures stdout/stderr from the subprocess via pipes, internal goroutines copy data from these pipes. `cmd.Wait()` blocks until both the process exits AND the pipe-copying goroutines finish. If the Python server spawns a subprocess that inherits the pipe file descriptors, `Wait()` will hang indefinitely even after the main process exits -- because the child still holds the pipe open. This is a documented Go issue, and it is worse on macOS where CommandContext has been reported to fail to kill processes.

**Prevention:**
- Set `cmd.WaitDelay` (Go 1.20+) to bound the time Wait spends after process exit
- Use `cmd.Cancel` function (Go 1.20+) to send SIGTERM instead of the default SIGKILL
- Redirect stdout/stderr to files instead of pipes to avoid the goroutine deadlock entirely
- Always consume pipe output in separate goroutines before calling Wait()
- Use process group kill as a backstop after WaitDelay expires

**Confidence:** HIGH -- documented in Go standard library docs and multiple Go GitHub issues.

**Sources:**
- [Go os/exec package documentation](https://pkg.go.dev/os/exec)
- [os/exec: consider changing Wait behavior](https://github.com/golang/go/issues/23019)
- [os/exec: CommandContext pipe close issue](https://github.com/golang/go/issues/21922)
- [os/exec: Unable to kill a command on macOS](https://github.com/golang/go/issues/27440)

### 6. Qwen3-TTS Audio Dropout and Quality Regressions

**The Problem:** mlx-audio has a documented bug where Qwen3-TTS generates audio with missing middle portions -- beginning and end are present but there is a gap/dropout in the middle. Additionally, recent updates have caused voice quality regressions (e.g., British accent voice clones reverting to American). The 0.6B model variant has been observed to produce complete silence for custom voices.

**Prevention:**
- Pin mlx-audio to a known-good version; do not auto-update
- Validate audio output: check duration > 0 and duration approximately matches expected length for input text
- Implement audio validation: check for silence detection (RMS energy below threshold) in generated output
- For critical text, split into shorter segments and concatenate to reduce dropout risk
- Keep a fallback TTS method (e.g., edge-tts) for when mlx-audio produces bad output

**Confidence:** HIGH -- documented in mlx-audio GitHub issues with reproduction steps.

**Sources:**
- [Qwen3-TTS audio dropout issue #464](https://github.com/Blaizzy/mlx-audio/issues/464)
- [Voice clone accent regression issue #439](https://github.com/Blaizzy/mlx-audio/issues/439)

---

## Technical Debt Patterns

### Version Pinning Trap
mlx-audio is under rapid development (57 open issues, frequent releases). Not pinning versions means behavior changes silently. But pinning too aggressively means missing critical bug fixes. **Strategy:** Pin minor version, allow patch updates, test on update.

### "It Works On My Machine" Model Cache
Models are downloaded to `~/.cache/huggingface/hub/` on first use. If the cache is corrupted (partial download, disk full), the server will fail to start with opaque errors. The `mlx-knife` tool provides `mlxk health --json` for cache health checks and `mlxk rm` for corrupted model cleanup.

### Dual Process Lifecycle
The Go CLI has its own lifecycle; the Python server has its own. Mismatches create debt:
- CLI crashes but server keeps running (orphan server consuming resources)
- Server crashes but CLI thinks it is running (stale PID tracking)
- User Ctrl+C kills CLI but not the server process group

### Implicit Health Assumptions
The mlx-audio server has `/v1/models` endpoint but no explicit `/health` or `/ready` endpoint. Checking "is the process running" is not the same as "is the model loaded and ready to serve." The server may be alive but still loading the model.

---

## Integration Gotchas (mlx-audio Specific)

### Server Startup Command
```
mlx_audio.server --host 127.0.0.1 --port 8778
```
The server is a FastAPI/uvicorn application. It loads models lazily on first request for a given model name, NOT at startup. This means the readiness probe must include a model loading request, not just a TCP connect.

### API is OpenAI-Compatible (Mostly)
The API follows OpenAI TTS spec (`POST /v1/audio/speech`) but:
- Model names are HuggingFace paths (e.g., `mlx-community/Qwen3-TTS-12Hz-1.7B-Base-bf16`), not simple names like "tts-1"
- The `voice` parameter behavior differs from OpenAI; custom voice requires specific reference audio setup
- Error responses may not match OpenAI error schema exactly
- Streaming support may behave differently or not be fully implemented

### Model Loading on First Request
The first TTS request after server start triggers model download (if not cached) AND model loading into GPU memory. This can take 10-30s. Subsequent requests to the same model are fast. Loading a different model unloads the previous one.

### Token Length Limit
Qwen3-TTS has a maximum token limit of ~1,600 tokens for the instruction/text parameter. Long text will be truncated or fail silently. You MUST implement client-side text chunking with sentence-boundary splitting for any text longer than ~200 words.

### Reference Audio Loading Bug
There was a reported issue where `ref_audio` would not load prior to inference when using the server API. Check the mlx-audio version for this fix if using voice cloning features.

### Dependency Hell
mlx-audio has fragile dependencies:
- `fastrtc[stt]==0.0.31` was yanked from PyPI, breaking `mlx-audio[all]` installs
- Python version constraints: some features require Python >= 3.10, web interface may not work with all versions
- uv sync failures reported on Python 3.12.9

**Strategy:** Use a dedicated virtual environment. Install mlx-audio in isolation. Test the install step as part of your setup process.

---

## Performance Traps

### Non-Determinism in MLX Output
MLX on Apple Silicon is non-deterministic even with temperature=0 and identical inputs. Different outputs occur between runs due to how Metal performs matrix operations. This means:
- Audio quality varies run-to-run for the same text
- Cannot use output hash for caching validation
- User may hear different prosody/quality for the same sentence in different sessions

**Source:** [The Hidden Problem With MLX: Non-Determinism](https://adityakarnam.com/mlx-non-determinism-apple-silicon/) (Confidence: HIGH)

### Low GPU Utilization
Qwen3-TTS shows ~12% GPU utilization during inference, resulting in RTF (Real-Time Factor) of ~x3 (1 minute of audio takes 3 minutes to generate). On MLX, the bottleneck may be CPU-side token generation rather than GPU-side audio synthesis. This is a known limitation of the current MLX implementation.

**Source:** [HuggingFace discussion on low GPU utilization](https://huggingface.co/Qwen/Qwen3-TTS-12Hz-1.7B-CustomVoice/discussions/18) (Confidence: MEDIUM -- may be CUDA-specific, MLX performance may differ)

### Thermal Throttling on Sustained Use
While the Mac Mini M4 has better thermal management than MacBook, sustained 30-60 minute TTS sessions can still cause thermal throttling. MLX benchmarks show that fewer, larger kernels reduce thermal stress. Monitor with: `sudo powermetrics --samplers smc -i 5000 | grep -i temp`

**Source:** [Building a macOS thermal throttling app](https://stanislas.blog/2025/12/macos-thermal-throttling-app/) (Confidence: MEDIUM -- Mac Mini M4 is less likely to throttle than laptops)

### First-Request Latency vs Subsequent Requests
Expect a bimodal latency distribution:
- First request: 10-40s (model load + shader compilation + inference)
- Subsequent requests: 1-10s (inference only, depending on text length)
- After sleep/wake: possibly back to first-request latency if memory was reclaimed

---

## macOS-Specific Issues

### Sleep/Wake Disrupts Everything
When the Mac sleeps:
- Network sockets may be invalidated (ECONNRESET, ECONNABORTED on localhost)
- The ML server process may be suspended by the kernel (SIGSTOP)
- On wake, the process resumes but sockets are dead
- If memory pressure was high before sleep, jetsam may have killed the process during sleep
- `pmset -g assertions` shows what prevents sleep; the ML server will NOT prevent sleep

**Mitigation:** After detecting wake (monitor `IOPMAssertionNotify` or poll process health), check if server is responsive. If not, restart it. Use `caffeinate -i` during active sessions to prevent sleep.

### Jetsam (macOS OOM Killer)
Unlike Linux's OOM killer, macOS jetsam prioritizes foreground GUI apps. Background daemon processes (like our TTS server) are prime targets for termination under memory pressure. There is no warning and no graceful shutdown -- the process simply disappears.

**Mitigation:** Monitor system memory pressure via `sysctl vm.memory_pressure_level` or parse `memory_pressure` output. Proactively restart the server with reduced memory footprint if system is under pressure.

### macOS Firewall and Gatekeeper
- First launch of unsigned Python binary may trigger Gatekeeper dialog
- Binding to 0.0.0.0 triggers firewall accept/deny dialog on every launch
- If user clicks "Deny," the server starts but cannot accept connections, failing silently
- Always use `--host 127.0.0.1` to avoid firewall prompts entirely

### LaunchD and Adaptive Scheduling
macOS uses adaptive CPU throttling for background processes. If the TTS server is not the frontmost application (it never will be), macOS may throttle its CPU allocation. Use `ProcessType = Interactive` in a launchd plist if running as a daemon, or use App Nap prevention via `NSProcessInfo.processInfo.beginActivity()`.

---

## "Looks Done But Isn't" Checklist

- [ ] **Server is "running" but model isn't loaded:** Process is alive, port is open, but first TTS request will take 20s because model loads lazily. You must warmup.
- [ ] **Port is "free" but in TIME_WAIT:** `lsof` shows nothing, but bind fails. Need SO_REUSEADDR or wait 1-4 minutes.
- [ ] **Process is "killed" but children survive:** Sent SIGTERM to PID, but uvicorn workers or Metal shader processes are still running. Must kill process group.
- [ ] **Audio file is "generated" but has silence/dropout:** File exists and has non-zero size, but middle portion is missing. Must validate audio content, not just file existence.
- [ ] **Server "responds" after sleep but is broken:** TCP connection succeeds but inference fails with Metal errors because GPU state was invalidated. Must do a health check with actual inference, not just a TCP ping.
- [ ] **Model is "cached" but corrupted:** HuggingFace cache has partial download. Server starts but crashes on model load with opaque tensor error. Use `mlx-knife health` to validate.
- [ ] **Install "succeeded" but wrong version:** pip installed but dependency was yanked or conflicting. Server imports fine but specific model support is broken. Test end-to-end TTS in install verification.
- [ ] **Text is "sent" but was truncated:** Input exceeds 1,600 token limit. Server doesn't error; just generates partial audio. Must split text client-side.
- [ ] **Graceful shutdown "worked" but Wait() hangs:** Go's `cmd.Wait()` blocks because a child process inherited pipe file descriptors. Set WaitDelay as a backstop.
- [ ] **Memory is "fine" according to Activity Monitor:** But unified memory includes GPU allocation which grows over time. The "Memory Used" column doesn't reflect Metal buffer allocations accurately.

---

## Recovery Strategies

### Stale Server Recovery (on startup)
```bash
# Find and kill any existing mlx-audio processes
lsof -ti:8778 | xargs kill -9 2>/dev/null
# Also check by process name
pgrep -f "mlx_audio.server" | xargs kill -9 2>/dev/null
# Wait for port to be released
sleep 1
```

### Mid-Session Server Crash Recovery
1. Detect crash via health check failure (HTTP request to `/v1/models` times out or connection refused)
2. Kill any remaining process group members
3. Wait for port release (up to 5s with polling)
4. Restart server
5. Wait for readiness (poll `/v1/models`)
6. Send warmup request
7. Resume operation

### Memory Pressure Recovery
1. Monitor RSS of server process
2. If RSS > threshold (e.g., 4GB for a 1.7B model that should use ~2-3GB):
   a. Complete any in-flight request
   b. Graceful shutdown (SIGTERM + wait)
   c. Restart server
3. If system memory pressure is CRITICAL:
   a. Shut down server immediately
   b. Notify user that TTS is temporarily unavailable
   c. Wait for memory pressure to subside
   d. Restart when safe

### Corrupted Model Cache Recovery
```bash
# Check cache health
python -m mlx_knife health --json

# If corrupted, remove and re-download
rm -rf ~/.cache/huggingface/hub/models--mlx-community--Qwen3-TTS-*

# Re-download will happen automatically on next server start + first request
```

### Go Process Manager Skeleton
```go
// Key fields for subprocess management
type ServerManager struct {
    cmd       *exec.Cmd
    pgid      int
    port      int
    healthy   bool
    startTime time.Time
}

// Critical: Set process group for clean shutdown
cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

// Critical: Set WaitDelay to prevent Wait() hang
cmd.WaitDelay = 10 * time.Second

// Critical: Set Cancel to send SIGTERM, not SIGKILL
cmd.Cancel = func() error {
    return cmd.Process.Signal(syscall.SIGTERM)
}

// Critical: Kill process GROUP, not just process
func (m *ServerManager) ForceKill() {
    syscall.Kill(-m.pgid, syscall.SIGKILL)
}
```

---

## Source Summary

| Source | Confidence | Category |
|--------|-----------|----------|
| [Go os/exec documentation](https://pkg.go.dev/os/exec) | HIGH | Pipe deadlock, WaitDelay |
| [mlx-audio GitHub issues](https://github.com/Blaizzy/mlx-audio/issues) | HIGH | Audio dropout, accent regression, dependency issues |
| [MLX memory leak issue #1124](https://github.com/ml-explore/mlx-examples/issues/1124) | HIGH | Memory growth during sustained use |
| [MLX memory leak issue #755](https://github.com/ml-explore/mlx/issues/755) | HIGH | Metal/MPS memory growth |
| [Uvicorn orphan processes issue #2289](https://github.com/Kludex/uvicorn/issues/2289) | HIGH | Child process cleanup failure |
| [Go process group killing](https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773) | HIGH | Process group management |
| [macOS port 5000 AirPlay conflict](https://nono.ma/port-5000-used-by-control-center-in-macos-ventura-controlce) | HIGH | Port conflict |
| [MLX non-determinism](https://adityakarnam.com/mlx-non-determinism-apple-silicon/) | HIGH | Output variability |
| [Qwen3-TTS audio dropout #464](https://github.com/Blaizzy/mlx-audio/issues/464) | HIGH | Audio quality |
| [Qwen3-TTS low GPU utilization](https://huggingface.co/Qwen/Qwen3-TTS-12Hz-1.7B-CustomVoice/discussions/18) | MEDIUM | May be CUDA-specific |
| [macOS jetsam documentation](https://newosxbook.com/articles/MemoryPressure.html) | HIGH | OOM behavior |
| [MLX Metal JIT cold start](https://ml-explore.github.io/mlx/build/html/install.html) | HIGH | Startup latency |
| [macOS sleep/wake socket issues](https://developer.apple.com/forums/thread/85038) | MEDIUM | Localhost socket survival after sleep |
| [Go exec.CommandContext macOS issue #27440](https://github.com/golang/go/issues/27440) | HIGH | macOS kill failure |
| [Qwen3-TTS token limit (Alibaba docs)](https://www.alibabacloud.com/help/en/model-studio/qwen-tts) | MEDIUM | API limit, may differ for local model |
| [mlx-audio voice regression #439](https://github.com/Blaizzy/mlx-audio/issues/439) | HIGH | Version update risk |
| [Thermal throttling on macOS](https://stanislas.blog/2025/12/macos-thermal-throttling-app/) | MEDIUM | Mac Mini M4 is less susceptible |
| [vLLM shared memory leaks](https://github.com/vllm-project/vllm/issues/8803) | MEDIUM | Pattern applies to MLX servers |
