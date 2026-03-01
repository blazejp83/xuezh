package audio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/joshp123/xuezh/internal/xuezh/envelope"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
	"github.com/joshp123/xuezh/internal/xuezh/process"
)

// LocalTTSError is a classified error from the local mlx-audio TTS backend.
// The Reason field distinguishes failure modes so the agent can take appropriate
// action (e.g., restart server, warn user, retry).
type LocalTTSError struct {
	Reason  string         // "server_down", "stale_pid", "oom", "timeout", "port_conflict", "text_too_long"
	Message string         // Human-readable description
	Details map[string]any // Structured details for the error envelope
}

func (e LocalTTSError) Error() string {
	return fmt.Sprintf("BACKEND_FAILED:%s:%s", e.Reason, e.Message)
}

// classifyHTTPError inspects an HTTP error (from client.Post) and classifies
// it into one of the LocalTTSError failure modes.
func classifyHTTPError(err error, port, pid int) LocalTTSError {
	details := map[string]any{"port": port, "pid": pid}

	// Timeout: context deadline exceeded or net/http timeout.
	if os.IsTimeout(err) || isTimeout(err) {
		return LocalTTSError{
			Reason:  "timeout",
			Message: "HTTP request timed out",
			Details: map[string]any{"port": port, "timeout_seconds": 120},
		}
	}

	// Connection refused: distinguish port_conflict vs stale_pid.
	if isConnectionRefused(err) {
		if pid > 0 && processAlive(pid) {
			return LocalTTSError{
				Reason:  "port_conflict",
				Message: "connection refused but server process is alive — port mismatch or another process took the port",
				Details: details,
			}
		}
		cleanStateFiles()
		return LocalTTSError{
			Reason:  "stale_pid",
			Message: "connection refused and server process is dead",
			Details: details,
		}
	}

	// Connection reset (process died during request) — likely OOM/crash.
	if isConnectionReset(err) {
		return LocalTTSError{
			Reason:  "oom",
			Message: "connection reset — server process may have been killed (OOM or crash)",
			Details: map[string]any{"port": port, "model": readModelFile()},
		}
	}

	// Fallback: server_down.
	return LocalTTSError{
		Reason:  "server_down",
		Message: fmt.Sprintf("local TTS request failed: %v", err),
		Details: details,
	}
}

// classifyHTTPResponse inspects a non-200 HTTP response and classifies it.
func classifyHTTPResponse(statusCode int, respBody []byte, port int) LocalTTSError {
	bodyStr := string(respBody)
	bodyPreview := bodyStr
	if len(bodyPreview) > 1024 {
		bodyPreview = bodyPreview[:1024]
	}

	// HTTP 400/422 with input-too-long indicators.
	if statusCode == http.StatusBadRequest || statusCode == 422 {
		lower := strings.ToLower(bodyStr)
		if strings.Contains(lower, "too long") || strings.Contains(lower, "too large") ||
			strings.Contains(lower, "max length") || strings.Contains(lower, "exceeds") {
			return LocalTTSError{
				Reason:  "text_too_long",
				Message: fmt.Sprintf("server rejected input (HTTP %d): text too long", statusCode),
				Details: map[string]any{"port": port, "http_status": statusCode, "body_preview": bodyPreview},
			}
		}
	}

	// HTTP 500 with OOM/memory/killed indicators.
	if statusCode == http.StatusInternalServerError {
		lower := strings.ToLower(bodyStr)
		if strings.Contains(lower, "memory") || strings.Contains(lower, "oom") || strings.Contains(lower, "killed") {
			return LocalTTSError{
				Reason:  "oom",
				Message: fmt.Sprintf("server returned OOM/memory error (HTTP %d)", statusCode),
				Details: map[string]any{"port": port, "model": readModelFile(), "http_status": statusCode, "body_preview": bodyPreview},
			}
		}
	}

	// Default: server_down.
	return LocalTTSError{
		Reason:  "server_down",
		Message: fmt.Sprintf("server returned HTTP %d: %s", statusCode, bodyPreview),
		Details: map[string]any{"port": port, "http_status": statusCode, "body_preview": bodyPreview},
	}
}

// isTimeout checks if an error is a timeout (context deadline or net/http timeout).
func isTimeout(err error) bool {
	if err == context.DeadlineExceeded {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}

// isConnectionRefused checks if an error is a "connection refused" error.
func isConnectionRefused(err error) bool {
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return strings.Contains(opErr.Err.Error(), "connection refused")
	}
	return strings.Contains(err.Error(), "connection refused")
}

// isConnectionReset checks if an error is a "connection reset" error.
func isConnectionReset(err error) bool {
	return strings.Contains(err.Error(), "connection reset")
}

// LocalTTS synthesises speech via HTTP POST to the running mlx-audio server.
// The raw WAV response is converted to the target format (derived from outPath
// extension, defaulting to "ogg") using ffmpeg.
//
// All TTS server errors are returned as LocalTTSError with a classified reason
// so the agent can take appropriate action. Non-server errors (path resolution,
// ffmpeg) are returned as their original error types.
func LocalTTS(text, voice, outPath, purpose string) (AudioResult, error) {
	// 1. Read port from state file.
	port := readPortFile()
	if port == 0 {
		return AudioResult{}, LocalTTSError{
			Reason:  "server_down",
			Message: "server not running (no port file)",
			Details: map[string]any{"port": 0, "pid": 0},
		}
	}

	// 2. Read model from state file.
	model := readModelFile()

	// 3. Verify server process is alive.
	pid := readPIDFile()
	if pid == 0 {
		cleanStateFiles()
		return AudioResult{}, LocalTTSError{
			Reason:  "server_down",
			Message: "server not running (no PID file)",
			Details: map[string]any{"port": port, "pid": 0},
		}
	}
	if !processAlive(pid) {
		cleanStateFiles()
		return AudioResult{}, LocalTTSError{
			Reason:  "stale_pid",
			Message: "server process is dead (stale PID)",
			Details: map[string]any{"port": port, "pid": pid},
		}
	}

	// 4. Default voice to "Vivian" if empty.
	if voice == "" {
		voice = "Vivian"
	}

	// 5. Resolve output path and create parent dirs.
	resolvedOut, err := paths.ResolveInWorkspace(outPath)
	if err != nil {
		return AudioResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(resolvedOut), 0o755); err != nil {
		return AudioResult{}, err
	}

	// 6. Ensure ffmpeg is available.
	if _, err := process.EnsureTool("ffmpeg"); err != nil {
		return AudioResult{}, err
	}

	// 7. Create temp file path for raw server response.
	tempPath := filepath.Join(filepath.Dir(resolvedOut), ".local-tts-"+uuid.New().String()+".wav")

	// 8. HTTP POST to mlx-audio server.
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/audio/speech", port)
	body := fmt.Sprintf(`{"model": %q, "input": %q, "voice": %q}`, model, text, voice)
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewBufferString(body))
	if err != nil {
		return AudioResult{}, classifyHTTPError(err, port, pid)
	}
	defer resp.Body.Close()

	// 9. Check HTTP status — read body (limited to 1KB for preview) and classify.
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return AudioResult{}, classifyHTTPResponse(resp.StatusCode, respBody, port)
	}

	// 10. Write response body to temp WAV file.
	rawBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return AudioResult{}, fmt.Errorf("failed to read TTS response body: %w", err)
	}
	if err := os.WriteFile(tempPath, rawBytes, 0o644); err != nil {
		return AudioResult{}, fmt.Errorf("failed to write temp WAV file: %w", err)
	}
	defer func() {
		_ = os.Remove(tempPath)
	}()

	// 11. Determine output format from outPath extension (default "ogg").
	fmtOut := strings.TrimPrefix(strings.ToLower(filepath.Ext(resolvedOut)), ".")
	if fmtOut == "" {
		fmtOut = "ogg"
	}
	if _, ok := supportedFormats[fmtOut]; !ok {
		fmtOut = "ogg"
	}

	// 12. Convert temp WAV to target format via ffmpeg.
	convertCmd, err := buildConvertCommand(tempPath, resolvedOut, fmtOut)
	if err != nil {
		return AudioResult{}, err
	}
	if _, err := process.RunChecked(convertCmd); err != nil {
		return AudioResult{}, err
	}

	// 13. Build artifact.
	artifact, err := artifactFor(resolvedOut, fmtOut, purpose)
	if err != nil {
		return AudioResult{}, err
	}

	// 14. Return AudioResult.
	data := map[string]any{
		"text":  text,
		"voice": voice,
		"out":   artifact.Path,
		"model": model,
		"backend": map[string]any{
			"id":       "local",
			"features": []string{"tts"},
		},
	}
	return AudioResult{Data: data, Artifacts: []envelope.Artifact{artifact}}, nil
}
