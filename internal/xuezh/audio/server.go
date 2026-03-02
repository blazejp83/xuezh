package audio

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/paths"
	"github.com/joshp123/xuezh/internal/xuezh/process"
)

// ServerState represents the current state of the mlx-audio TTS server.
type ServerState struct {
	Status         string  // "ready", "starting", "stopped", "error"
	Port           int
	PID            int
	Model          string
	StartupSeconds float64 // time from spawn to ready
}

// StopResult represents the outcome of a server stop operation.
type StopResult struct {
	Status     string // "stopped"
	WasRunning bool   // true if a server was actually running and was stopped
}

// StatusResult represents the current server status.
type StatusResult struct {
	Status        string  // "running" or "stopped"
	Port          int     // 0 if stopped
	PID           int     // 0 if stopped
	Model         string  // "" if stopped
	UptimeSeconds float64 // 0 if stopped
}

// serverDir returns the path to {workspace}/server/, creating it if needed.
func serverDir() (string, error) {
	workspace, err := paths.EnsureWorkspace()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(workspace, "server")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// pidPath returns the path to the PID file.
func pidPath() (string, error) {
	dir, err := serverDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "pid"), nil
}

// portPath returns the path to the port file.
func portPath() (string, error) {
	dir, err := serverDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "port"), nil
}

// modelPath returns the path to the model file.
func modelPath() (string, error) {
	dir, err := serverDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "model"), nil
}

// logPath returns the path to the server log file.
func logPath() (string, error) {
	dir, err := serverDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "server.log"), nil
}

// cleanStateFiles removes PID, port, and model files.
func cleanStateFiles() {
	if p, err := pidPath(); err == nil {
		_ = os.Remove(p)
	}
	if p, err := portPath(); err == nil {
		_ = os.Remove(p)
	}
	if p, err := modelPath(); err == nil {
		_ = os.Remove(p)
	}
}

// readPIDFile reads the PID from the PID file. Returns 0 if not found or invalid.
func readPIDFile() int {
	p, err := pidPath()
	if err != nil {
		return 0
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0
	}
	return pid
}

// readPortFile reads the port from the port file. Returns 0 if not found or invalid.
func readPortFile() int {
	p, err := portPath()
	if err != nil {
		return 0
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return 0
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || port <= 0 {
		return 0
	}
	return port
}

// readModelFile reads the model name from the model file. Returns "" if not found.
func readModelFile() string {
	p, err := modelPath()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// processAlive checks if a process with the given PID is alive.
func processAlive(pid int) bool {
	return syscall.Kill(pid, 0) == nil
}

// cleanStaleServer checks for a stale server process and cleans up.
// Returns an error if a live server is already running and responding.
func cleanStaleServer(port int) error {
	pid := readPIDFile()
	if pid == 0 {
		// No PID file — nothing to clean.
		return nil
	}

	if !processAlive(pid) {
		// Process is dead (ESRCH). Clean up stale files.
		cleanStateFiles()
		return nil
	}

	// Process is alive. Check if it's responding on the expected port.
	storedPort := readPortFile()
	if storedPort == 0 {
		storedPort = port
	}

	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/models", storedPort)
	resp, err := client.Get(url)
	if err == nil {
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return fmt.Errorf("server already running (pid=%d, port=%d)", pid, storedPort)
		}
	}

	// Process alive but not responding — kill the stale process group.
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	// Brief wait for process to terminate.
	time.Sleep(100 * time.Millisecond)
	cleanStateFiles()
	return nil
}

// readLogTail reads the last N lines from the server log file.
func readLogTail(n int) string {
	lp, err := logPath()
	if err != nil {
		return ""
	}
	f, err := os.Open(lp)
	if err != nil {
		return ""
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// waitForReady polls the server health endpoint until it responds with 200.
func waitForReady(port, pid int) error {
	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/models", port)
	deadline := time.Now().Add(120 * time.Second)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check if process is still alive.
			if !processAlive(pid) {
				tail := readLogTail(20)
				cleanStateFiles()
				return fmt.Errorf("BACKEND_FAILED:crash:server process died during startup\nlog_tail:\n%s", tail)
			}

			resp, err := client.Get(url)
			if err == nil {
				_ = resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					return nil
				}
			}

			if time.Now().After(deadline) {
				// Timeout — kill the process group and clean up.
				_ = syscall.Kill(-pid, syscall.SIGKILL)
				cleanStateFiles()
				return fmt.Errorf("BACKEND_FAILED:timeout:server did not become ready within 120 seconds")
			}
		}
	}
}

// warmUp sends a warm-up TTS request to pre-load the model.
func warmUp(port int, model string) {
	client := &http.Client{Timeout: 60 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/audio/speech", port)
	body := fmt.Sprintf(`{"model": %q, "input": "你好", "voice": "Vivian"}`, model)
	resp, err := client.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		// Warm-up failure is non-fatal — log but don't fail start.
		return
	}
	defer resp.Body.Close()
	// Drain the response body (raw audio bytes).
	_, _ = io.Copy(io.Discard, resp.Body)
}

// StopServer stops the running mlx-audio server with a three-phase graceful
// shutdown: SIGTERM to process group, wait, then SIGKILL if needed.
func StopServer() (StopResult, error) {
	pid := readPIDFile()
	if pid == 0 {
		// No PID file — server not running.
		return StopResult{Status: "stopped", WasRunning: false}, nil
	}

	if !processAlive(pid) {
		// Process is dead — clean up stale files.
		cleanStateFiles()
		return StopResult{Status: "stopped", WasRunning: false}, nil
	}

	// Phase 1: SIGTERM to process group.
	err := syscall.Kill(-pid, syscall.SIGTERM)
	if err != nil {
		if errors.Is(err, syscall.ESRCH) {
			cleanStateFiles()
			return StopResult{Status: "stopped", WasRunning: false}, nil
		}
		if errors.Is(err, syscall.EPERM) {
			return StopResult{}, fmt.Errorf("permission denied killing process group %d", pid)
		}
	}

	// Wait up to 5 seconds for process to die.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			cleanStateFiles()
			return StopResult{Status: "stopped", WasRunning: true}, nil
		}
		time.Sleep(250 * time.Millisecond)
	}

	// Phase 2: SIGKILL to process group.
	err = syscall.Kill(-pid, syscall.SIGKILL)
	if err != nil && !errors.Is(err, syscall.ESRCH) {
		// Best effort — continue to cleanup.
	}

	// Wait up to 2 seconds for forced kill.
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}

	// Phase 3: Clean up state files (never remove server.log).
	cleanStateFiles()
	return StopResult{Status: "stopped", WasRunning: true}, nil
}

// ServerStatus returns the current status of the mlx-audio server.
func ServerStatus() (StatusResult, error) {
	pid := readPIDFile()
	if pid == 0 {
		return StatusResult{Status: "stopped"}, nil
	}

	if !processAlive(pid) {
		// Stale PID — clean up.
		cleanStateFiles()
		return StatusResult{Status: "stopped"}, nil
	}

	port := readPortFile()
	model := readModelFile()

	// Verify server is actually responsive.
	if port > 0 {
		client := &http.Client{Timeout: 2 * time.Second}
		url := fmt.Sprintf("http://127.0.0.1:%d/v1/models", port)
		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
		}
		// Whether responsive or not, the process is alive — report running.
	}

	// Calculate uptime from PID file modification time.
	var uptime float64
	pp, err := pidPath()
	if err == nil {
		if info, err := os.Stat(pp); err == nil {
			uptime = time.Since(info.ModTime()).Seconds()
		}
	}

	return StatusResult{
		Status:        "running",
		Port:          port,
		PID:           pid,
		Model:         model,
		UptimeSeconds: uptime,
	}, nil
}

// StartServer spawns the mlx-audio server as a managed subprocess with process
// group isolation, health polling, and a warm-up TTS request.
func StartServer(port int, model, pythonPath string) (ServerState, error) {
	// Ensure python is available at the configured path.
	if _, err := process.EnsureTool(pythonPath); err != nil {
		return ServerState{}, err
	}

	// Step 1: Clean stale server state.
	if err := cleanStaleServer(port); err != nil {
		return ServerState{}, err
	}

	// Step 2: Check port availability.
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return ServerState{}, fmt.Errorf("BACKEND_FAILED:port_conflict:port %d is already in use", port)
	}
	listener.Close()

	// Step 3: Spawn subprocess.
	startTime := time.Now()

	lp, err := logPath()
	if err != nil {
		return ServerState{}, err
	}
	logFile, err := os.OpenFile(lp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return ServerState{}, fmt.Errorf("failed to open log file: %w", err)
	}

	cmd := exec.Command(pythonPath, "-m", "mlx_audio.server",
		"--host", "127.0.0.1",
		"--port", strconv.Itoa(port),
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return ServerState{}, fmt.Errorf("BACKEND_FAILED:spawn_failed:failed to start mlx-audio server: %w", err)
	}

	// Close the log file handle — the child process has its own fd.
	logFile.Close()

	pid := cmd.Process.Pid

	// Step 4: Write state files.
	pp, _ := pidPath()
	_ = os.WriteFile(pp, []byte(strconv.Itoa(pid)), 0o644)

	ptp, _ := portPath()
	_ = os.WriteFile(ptp, []byte(strconv.Itoa(port)), 0o644)

	mp, _ := modelPath()
	_ = os.WriteFile(mp, []byte(model), 0o644)

	// Step 5: Health polling.
	if err := waitForReady(port, pid); err != nil {
		return ServerState{}, err
	}

	// Step 6: Warm-up TTS request.
	warmUp(port, model)

	// Step 7: Return ServerState.
	elapsed := time.Since(startTime).Seconds()
	return ServerState{
		Status:         "ready",
		Port:           port,
		PID:            pid,
		Model:          model,
		StartupSeconds: elapsed,
	}, nil
}
