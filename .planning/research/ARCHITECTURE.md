# Two-Process Architecture Research: Go CLI + Python ML Server

**Research Date:** 2026-02-28
**Domain:** CLI tool (xuezh) managing a local TTS ML model server (mlx-audio)
**Target Platform:** macOS ARM64 (Mac Mini M4)

---

## System Overview

```
    Agent (Clawdbot)
         |
         | invokes CLI
         v
  +------------------+          HTTP localhost:PORT          +---------------------+
  |   xuezh (Go)     | -------------------------------------> | mlx-audio (Python)  |
  |                  |          POST /v1/audio/speech        |                     |
  | - CLI dispatch   | <------------------------------------- | - Qwen3-TTS 1.7B   |
  | - PID management |          audio/wav response           | - MLX framework     |
  | - Health polling |                                       | - ~2-3 GB RAM       |
  | - Lifecycle ctrl |                                       | - OpenAI-compat API |
  +------------------+                                       +---------------------+
         |                                                          |
         | reads/writes                                             | holds in memory
         v                                                          v
  +------------------+                                       +---------------------+
  | Workspace Files  |                                       | ML Model Weights    |
  | - PID file       |                                       | - Loaded on first   |
  | - Port file      |                                       |   /v1/audio/speech   |
  | - Server logs    |                                       |   request or at     |
  | - Audio artifacts|                                       |   startup           |
  +------------------+                                       +---------------------+
```

**Lifecycle sequence:**

```
  Agent session start
       |
       v
  xuezh audio server start
       |
       +---> Spawn mlx_audio.server as subprocess
       +---> Write PID + port to workspace
       +---> Poll GET /v1/models until 200 OK
       +---> Return {ok: true, status: "ready", port: N}
       |
       v
  xuezh audio tts --text "..." --backend mlx-audio
       |
       +---> POST http://127.0.0.1:PORT/v1/audio/speech
       +---> Stream response to temp file
       +---> Convert with ffmpeg
       +---> Return artifact envelope
       |
       v
  xuezh audio server stop
       |
       +---> Read PID from file
       +---> SIGTERM, wait grace period
       +---> SIGKILL if still alive
       +---> Remove PID + port files
       +---> Return {ok: true, status: "stopped"}
```

---

## Component Responsibilities

| Component | Responsibility | Key Constraint |
|-----------|---------------|----------------|
| `cli/cli.go` | Route `audio server {start,stop,status}` commands | Must emit JSON envelopes, no interactive output |
| `audio/server.go` (new) | Server lifecycle: start, stop, status, health polling | Must handle stale PIDs, orphan cleanup |
| `audio/mlxaudio.go` (new) | TTS via HTTP to mlx-audio server | Must match existing `TTSAudio()` signature pattern |
| `process/process.go` (extended) | Background subprocess spawn, PID file ops | Must use process groups (`Setpgid`) to prevent orphans |
| `paths/paths.go` (extended) | Server state file paths (PID, port, logs) | Must live inside existing workspace tree |
| mlx-audio server (external) | Serve TTS via OpenAI-compatible HTTP API | Holds ~2-3 GB model in memory, 10-30s cold start |

---

## Recommended Process Management Approach

### 1. PID File Management

**Location:** Store PID and port files inside the existing workspace directory.

```
~/.clawdbot/workspace/xuezh/
  server/
    pid           # contains PID as plain text integer
    port          # contains port number as plain text integer
    server.log    # stdout/stderr from mlx-audio process
```

This aligns with the existing workspace convention (`paths.WorkspaceDir()`) and avoids
introducing new directory conventions. The workspace is already user-scoped and has
established cleanup patterns via `retention.go`.

**Why not XDG_RUNTIME_DIR or /tmp:**
- macOS has no XDG_RUNTIME_DIR by default [Source: XDG Base Directory Specification]
- `/tmp` files can be cleaned unpredictably by macOS
- The workspace is already the canonical state directory for this tool

**Stale PID detection pattern:**

```
1. Read PID from file
2. Call syscall.Kill(pid, 0)  -- signal 0 checks existence without killing
3. If ESRCH -> process dead, PID is stale, clean up files
4. If nil (process exists) -> verify it is actually our server:
   a. Read port file
   b. Try GET http://127.0.0.1:PORT/v1/models
   c. If responds -> server is running
   d. If connection refused -> stale PID (some other process reused the PID)
5. If EPERM -> process exists but we lack permission (unlikely for our own process)
```

**File locking:** Use `syscall.Flock()` on the PID file with `LOCK_EX | LOCK_NB` (exclusive,
non-blocking) during startup to prevent race conditions when two CLI invocations try to
start the server simultaneously. The lock is advisory but sufficient since all participants
(xuezh CLI instances) cooperate.

**Confidence:** HIGH -- This is the standard pattern used across Go process managers.
Sources: [gofrs/flock](https://github.com/gofrs/flock),
[bbengfort/x/pid](https://pkg.go.dev/github.com/bbengfort/x/pid),
[yoanbernabeu/grepai/daemon](https://pkg.go.dev/github.com/yoanbernabeu/grepai/daemon)

### 2. Port Management

**Recommendation: Fixed default port (8921) with override.**

Rationale for fixed port over dynamic:
- The mlx-audio server is started as a detached subprocess, not a child with a pipe --
  there is no easy channel to receive a dynamically assigned port back from the Python process
- A fixed port simplifies the mental model for debugging (`curl localhost:8921/v1/models`)
- Port conflicts are rare in single-user CLI contexts
- The port is written to a file anyway, so consumers read it from there

The port should be configurable via:
1. CLI flag: `--port 8921`
2. Config file: `[audio.server] port = 8921`
3. Environment: `XUEZH_TTS_SERVER_PORT=8921`
4. Default: `8921`

Port conflict detection at startup:

```
1. Attempt net.Listen("tcp", "127.0.0.1:PORT")
2. If error -> port in use
3. Immediately close the listener
4. Then start mlx-audio with --port PORT
```

Note: There is a TOCTOU gap between releasing the port and mlx-audio binding it. This is
acceptable for a single-user CLI tool where the race window is milliseconds.

**Confidence:** MEDIUM -- Dynamic port (`:0`) is technically superior but requires IPC to
communicate the assigned port back. Ollama uses dynamic ports because its runner is a child
process with controlled stdout. Our case has the server as a semi-detached subprocess, making
fixed port more pragmatic.
Sources: [Ollama llm/server.go](https://github.com/ollama/ollama/blob/main/llm/server.go),
[yellowduck.be - Dynamic ports in Go](https://www.yellowduck.be/posts/dynamically-allocating-ports-in-a-webserver-using-go)

### 3. Health Check Polling (Wait for Ready)

**Pattern: Linear poll with timeout, no exponential backoff.**

For a local server loading a model, exponential backoff is counterproductive -- the server
transitions from "loading" to "ready" in a single moment. We want to detect that moment
quickly.

```
Recommended polling parameters:
- Poll interval: 250ms (fast enough to detect readiness, low enough overhead for localhost)
- Overall timeout: 120s (Qwen3-TTS 1.7B could take 10-30s; 120s provides generous margin)
- Stall timeout: 60s (if no progress for 60s, assume failure)
```

**Health check endpoint options for mlx-audio:**
- Primary: `GET /v1/models` -- returns 200 when server is up and accepting requests
- Fallback: `GET /` -- may return 404 or root page, but a TCP connection success
  indicates the server process is listening

**Implementation pattern (inspired by Ollama):**

```
func waitForReady(ctx context.Context, port int, timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    ticker := time.NewTicker(250 * time.Millisecond)
    defer ticker.Stop()

    client := &http.Client{Timeout: 2 * time.Second}
    url := fmt.Sprintf("http://127.0.0.1:%d/v1/models", port)

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            if time.Now().After(deadline) {
                return fmt.Errorf("server did not become ready within %v", timeout)
            }
            resp, err := client.Get(url)
            if err != nil {
                continue  // server not yet listening
            }
            resp.Body.Close()
            if resp.StatusCode == http.StatusOK {
                return nil  // server is ready
            }
        }
    }
}
```

**Confidence:** HIGH -- Ollama uses the same pattern (poll /health with timeout). The 250ms
interval is standard for local health checks. Kubernetes uses similar patterns for readiness
probes.
Sources: [Ollama llm/server.go](https://github.com/ollama/ollama/blob/main/llm/server.go),
[Ollama Architecture](https://deepwiki.com/ollama/ollama/2-architecture),
[VictoriaMetrics - Graceful Shutdown in Go](https://victoriametrics.com/blog/go-graceful-shutdown/)

### 4. Subprocess Startup

**Use `exec.Command` with process group isolation:**

```go
cmd := exec.Command("python3", "-m", "mlx_audio.server",
    "--host", "127.0.0.1",
    "--port", strconv.Itoa(port),
)

// Create new process group so SIGTERM to CLI does not propagate to server
cmd.SysProcAttr = &syscall.SysProcAttr{
    Setpgid: true,
    Pgid:    0,  // new group with PID as PGID
}

// Redirect stdout/stderr to log file
logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
cmd.Stdout = logFile
cmd.Stderr = logFile

cmd.Start()  // non-blocking
// Write cmd.Process.Pid to PID file
```

**Why Setpgid:** Without it, if the CLI process receives SIGINT (Ctrl+C), the signal
propagates to the child process group, killing the server. With `Setpgid: true`, the server
lives in its own process group and persists after the CLI exits. This is essential because the
server must survive across multiple CLI invocations during a learning session.

**Why NOT full daemonization:** Modern consensus is that daemonization (double-fork, setsid,
etc.) is an anti-pattern for Go programs. The process group approach gives us the persistence
we need without the complexity of daemon semantics.

**Confidence:** HIGH -- Process group isolation is the standard Go pattern for managed
subprocesses that should outlive their parent.
Sources: [Killing child processes in Go](https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773),
[Avoid Go orphan processes](https://www.sobyte.net/post/2021-08/avoid-go-command-orphan-processes/),
[Daemonization is an Anti-Pattern](https://medium.com/@yuseferi/daemonization-is-an-anti-pattern-using-os-native-supervision-for-go-binaries-599dbdab18cd)

### 5. Graceful Shutdown

**Three-phase shutdown pattern:**

```
Phase 1: SIGTERM (polite request)
  - Send syscall.Kill(pid, syscall.SIGTERM)
  - Wait up to 5 seconds
  - Check if process exited (syscall.Kill(pid, 0) returns ESRCH)

Phase 2: SIGKILL (force)
  - If still alive after 5s, send syscall.Kill(pid, syscall.SIGKILL)
  - Wait up to 2 seconds

Phase 3: Cleanup
  - Remove PID file
  - Remove port file
  - Log outcome
```

**Why SIGTERM first:** The Python server may be in the middle of a TTS generation. SIGTERM
allows it to finish the current request and release resources cleanly. SIGKILL is the fallback
for hung processes.

**Why 5s grace period:** TTS generation for typical Chinese sentences (5-20 characters) takes
1-3 seconds. A 5s grace period covers completion of an in-flight request.

Note: Ollama uses direct `SIGKILL` without a SIGTERM phase because their runner lifetime is
tied to a single request. Our case is different -- the server persists and may be serving a
request when shutdown is requested.

**Confidence:** HIGH -- SIGTERM then SIGKILL with timeout is the universal graceful shutdown
pattern on Unix systems.
Sources: [VictoriaMetrics - Graceful Shutdown](https://victoriametrics.com/blog/go-graceful-shutdown/),
[FreeCodeCamp - Graceful Shutdowns in Go](https://www.freecodecamp.org/news/graceful-shutdowns-k8s-go/),
[Rafal Roppel - Graceful Shutdown Explained](https://rafalroppel.medium.com/graceful-shutdown-in-go-explained-signals-contexts-and-the-correct-shutdown-sequence-f24fd9ef8fac)

### 6. Log Management

**Redirect subprocess stdout/stderr to a single log file:**

```
~/.clawdbot/workspace/xuezh/server/server.log
```

**Log rotation:** Not needed for this use case. The server runs for minutes to hours during a
learning session, not days. The log file is overwritten on each `server start` (use
`os.O_CREATE|os.O_WRONLY|os.O_TRUNC` instead of `O_APPEND`). This prevents unbounded growth
and keeps the most recent session's logs available for debugging.

**Log tail for status:** The `audio server status` command should read the last N lines of the
log file and include them in the response envelope for agent diagnostics.

**Confidence:** MEDIUM -- Truncation on restart is simpler than rotation but loses history.
Acceptable for single-user CLI tool where only the current session matters.
Sources: [Go Process Management - Calmops](https://calmops.com/programming/golang/go-process-management-subprocess/),
[Running Go Applications in Background](https://blog.questionable.services/article/running-go-applications-in-the-background/)

---

## Data Flow

### TTS Request Flow (with server running)

```
Agent                    xuezh CLI                 mlx-audio server
  |                         |                           |
  |  audio tts --text "..."  |                           |
  |  --backend mlx-audio     |                           |
  | -----------------------> |                           |
  |                         |  Read port from file       |
  |                         |  POST /v1/audio/speech     |
  |                         |  {model, input, voice}     |
  |                         | ----------------------->   |
  |                         |                           |  Generate audio
  |                         |                           |  (1-3 seconds)
  |                         |   <-----------------------  |
  |                         |   audio/wav response       |
  |                         |                           |
  |                         |  Write temp file           |
  |                         |  ffmpeg convert to target  |
  |                         |  Store artifact            |
  |                         |                           |
  |  <--------------------- |                           |
  |  JSON envelope:          |                           |
  |  {ok, data, artifacts}   |                           |
```

### Server Lifecycle Flow

```
Agent                    xuezh CLI                 OS / mlx-audio
  |                         |                           |
  |  audio server start     |                           |
  | -----------------------> |                           |
  |                         |  Check stale PID           |
  |                         |  Acquire flock on PID file |
  |                         |  Check port available      |
  |                         |  exec.Command(python3 ...) |
  |                         |  Write PID, port files     |
  |                         |  Poll GET /v1/models       |
  |                         |  .... 250ms ....           |
  |                         |  .... 250ms ....           |
  |                         |  .... 250ms .... (10-30s)  |
  |                         |  200 OK                    |
  |  <--------------------- |                           |
  |  {ok, status: "ready",   |                           |
  |   port: 8921, pid: N}   |                           |
  |                         |                           |
  |  ... learning session .. |                           |
  |                         |                           |
  |  audio server stop      |                           |
  | -----------------------> |                           |
  |                         |  Read PID from file        |
  |                         |  SIGTERM -> wait 5s        |
  |                         |  (SIGKILL if needed)       |
  |                         |  Remove PID, port files    |
  |  <--------------------- |                           |
  |  {ok, status: "stopped"} |                           |
```

---

## Integration Points (HTTP API Contract)

### mlx-audio Server Endpoints Used

| Endpoint | Method | Purpose | Request Body | Response |
|----------|--------|---------|-------------|----------|
| `/v1/models` | GET | Health check / readiness probe | None | 200 + JSON model list |
| `/v1/audio/speech` | POST | Generate TTS audio | `{model, input, voice, speed}` | Binary audio stream |

### TTS Request Format (OpenAI-compatible)

```json
{
  "model": "mlx-community/Qwen2.5-TTS-0.5B-bf16",
  "input": "你好世界",
  "voice": "Qwen2.5-TTS",
  "speed": 1.0
}
```

### TTS Response

- Content-Type: `audio/wav` (or other format depending on server config)
- Body: Raw audio bytes
- Errors: Standard HTTP status codes (500 for generation failure, 422 for bad input)

### CLI Envelope Contract (existing pattern, extended)

**Server start response:**
```json
{
  "ok": true,
  "schema_version": "0.9",
  "command": "audio.server.start",
  "data": {
    "status": "ready",
    "port": 8921,
    "pid": 12345,
    "startup_seconds": 14.2,
    "backend": {"id": "mlx-audio", "features": ["tts", "server"]}
  }
}
```

**Server status response:**
```json
{
  "ok": true,
  "schema_version": "0.9",
  "command": "audio.server.status",
  "data": {
    "status": "running",
    "port": 8921,
    "pid": 12345,
    "uptime_seconds": 342
  }
}
```

**Server stop response:**
```json
{
  "ok": true,
  "schema_version": "0.9",
  "command": "audio.server.stop",
  "data": {
    "status": "stopped",
    "was_running": true
  }
}
```

---

## Anti-Patterns to Avoid

### 1. Full Daemonization (double-fork)

**What:** Implementing Unix daemon semantics with double-fork, setsid, chdir /, close fds.

**Why avoid:** This is explicitly called out as an anti-pattern for Go applications. It
commits you to a long-term maintenance liability, managing orphan processes across restarts,
handling zombie reaping, and dealing with signal inheritance. The process group approach
(`Setpgid: true`) gives us subprocess persistence without daemon complexity.

**Source:** [Daemonization is an Anti-Pattern (Yusef Mohamadi, 2026)](https://medium.com/@yuseferi/daemonization-is-an-anti-pattern-using-os-native-supervision-for-go-binaries-599dbdab18cd)
-- Confidence: HIGH

### 2. PID File Without Validation

**What:** Trusting a PID file at face value without checking if the process is actually alive
and is actually the server we started.

**Why avoid:** PIDs are recycled by the OS. After a crash, the PID in the file may belong to
an unrelated process. Always validate with `kill(pid, 0)` + health check HTTP request.

**Source:** [bbengfort/x/pid](https://pkg.go.dev/github.com/bbengfort/x/pid),
[grepai daemon package](https://pkg.go.dev/github.com/yoanbernabeu/grepai/daemon)
-- Confidence: HIGH

### 3. Polling Without Timeout

**What:** Polling the health endpoint in an infinite loop with no upper bound.

**Why avoid:** If the server crashes during model loading, the CLI hangs forever. Ollama uses
a 10-minute timeout for large model loading. Our model is smaller, so 120 seconds is
sufficient. Always have both an overall timeout and a stall detector (no progress for N
seconds).

**Source:** [Ollama server.go](https://github.com/ollama/ollama/blob/main/llm/server.go)
-- Confidence: HIGH

### 4. Spawning Without Process Group Isolation

**What:** Using `cmd.Start()` without `Setpgid: true`, leaving the server in the CLI's
process group.

**Why avoid:** When the CLI exits (normally or via Ctrl+C), the server receives the same
signal and dies. This defeats the purpose of a persistent server. The xuezh CLI is invoked
repeatedly by the agent -- each invocation must not affect the running server.

**Source:** [Killing child processes in Go](https://medium.com/@felixge/killing-a-child-process-and-all-of-its-children-in-go-54079af94773)
-- Confidence: HIGH

### 5. Using cmd.Wait() for Detached Servers

**What:** Calling `cmd.Wait()` after `cmd.Start()` for a server that should persist.

**Why avoid:** `Wait()` blocks until the process exits. For a server that should outlive the
CLI invocation, we start the process and immediately return. The PID file is our handle to the
process for future `status` and `stop` commands.

**Source:** [GoLinuxCloud - Start process in background](https://www.golinuxcloud.com/golang-monitor-background-process/)
-- Confidence: HIGH

### 6. Dynamic Port Without IPC Channel

**What:** Starting the server on port 0 (OS-assigned) and trying to discover the assigned
port without a reliable communication channel.

**Why avoid:** The mlx-audio server does not write its port to stdout in a parseable format.
Without a pipe, socket, or shared file protocol with the server, there is no reliable way to
discover a dynamically assigned port. A fixed (configurable) port is pragmatic for this case.

**Source:** Original analysis. Ollama avoids this problem because it controls both the Go
parent and the runner binary, passing `--port` and reading it back via controlled stdout.
-- Confidence: MEDIUM

### 7. Ignoring Server Crash After Startup

**What:** Only checking server health at startup, then assuming it stays alive.

**Why avoid:** The server could crash (OOM, segfault in MLX, Python exception) mid-session.
The TTS request path should handle connection errors gracefully and return a meaningful error
envelope, not a cryptic connection refused message.

**Source:** Original analysis based on Ollama's processCompleted() goroutine pattern.
-- Confidence: MEDIUM

---

## Reference Implementations Studied

### Ollama (Go CLI + llama.cpp runner subprocess)

**Architecture:** Go scheduler manages multiple llama.cpp runner subprocesses. Each runner is
an HTTP server on an ephemeral port. The scheduler maintains a reference-counted pool with
idle expiration (default 5 minutes).

**Key patterns adopted:**
- Health polling via HTTP GET to /health endpoint
- Port passed as `--port` argument to subprocess
- Process reaping via goroutine waiting on `cmd.Wait()`
- Subprocess environment setup (library paths)

**Key differences from our case:**
- Ollama manages multiple concurrent runners; we have one server
- Ollama runners are short-lived (expire after idle); our server persists for a session
- Ollama uses SIGKILL directly (runner lifetime tied to scheduler); we need graceful SIGTERM
- Ollama controls both sides of the protocol; we use an external Python server

**Source:** [Ollama Architecture (DeepWiki)](https://deepwiki.com/ollama/ollama/2-architecture),
[ollama/llm/server.go](https://github.com/ollama/ollama/blob/main/llm/server.go)
-- Confidence: HIGH

### Jan (llama.cpp engine management)

**Architecture:** Desktop app managing llama.cpp as a local inference engine. Uses the same
pattern of subprocess management with health checks.

**Source:** [Jan llama.cpp engine docs](https://www.jan.ai/docs/llama-cpp-server)
-- Confidence: LOW (limited detail available)

### LocalAI (Go + multiple ML backends)

**Architecture:** Go server managing multiple ML backends as subprocesses. Includes a watchdog
mechanism for backends that can detect stalling.

**Key pattern of interest:** Backend watchdog -- a periodic health check that restarts crashed
backends automatically. Worth considering for robustness but likely overkill for our
single-server, session-scoped use case.

**Source:** [LocalAI distributed inference docs](https://localai.io/features/distribute/)
-- Confidence: LOW (watchdog details not publicly documented in detail)

---

## Summary of Recommendations

| Aspect | Recommendation | Confidence |
|--------|---------------|------------|
| PID file location | `~/.clawdbot/workspace/xuezh/server/pid` | HIGH |
| PID validation | `kill(pid, 0)` + HTTP health check | HIGH |
| Race prevention | `syscall.Flock()` on PID file | HIGH |
| Port strategy | Fixed default 8921, configurable | MEDIUM |
| Port conflict check | `net.Listen` then close before spawn | HIGH |
| Health polling | 250ms interval, 120s timeout, linear | HIGH |
| Health endpoint | `GET /v1/models` | MEDIUM |
| Subprocess isolation | `Setpgid: true` on exec.Command | HIGH |
| Graceful shutdown | SIGTERM (5s grace) then SIGKILL | HIGH |
| Log management | Single file, truncate on restart | MEDIUM |
| Daemonization | Do not daemonize; use process groups only | HIGH |

---

*Research completed: 2026-02-28*
*Sources verified through web search, code review, and documentation analysis*
