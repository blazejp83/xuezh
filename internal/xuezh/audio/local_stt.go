package audio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
	"github.com/joshp123/xuezh/internal/xuezh/jsonio"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
)

// DefaultSTTModel is the default ASR model for local STT via mlx-audio.
// Exported so callers (e.g., CLI) can reference it, but the CLI uses its own
// resolution chain (flag > config > env > default). This constant serves as
// a safety net for programmatic callers that pass an empty model string.
const DefaultSTTModel = "mlx-community/Qwen3-ASR-1.7B-8bit"

// LocalSTTError is a classified error from the local mlx-audio STT backend.
// The Reason field distinguishes failure modes so the agent can take appropriate
// action (e.g., restart server, warn user, retry).
type LocalSTTError struct {
	Reason  string         // "server_down", "stale_pid", "oom", "timeout", "port_conflict", "unsupported_format", "model_error"
	Message string         // Human-readable description
	Details map[string]any // Structured details for the error envelope
}

func (e LocalSTTError) Error() string {
	return fmt.Sprintf("BACKEND_FAILED:%s:%s", e.Reason, e.Message)
}

// classifySTTHTTPError inspects an HTTP error (from client.Do/Post) and classifies
// it into one of the LocalSTTError failure modes. Mirrors classifyHTTPError from local.go.
func classifySTTHTTPError(err error, port, pid int) LocalSTTError {
	details := map[string]any{"port": port, "pid": pid}

	// Timeout: context deadline exceeded or net/http timeout.
	if os.IsTimeout(err) || isTimeout(err) {
		return LocalSTTError{
			Reason:  "timeout",
			Message: "HTTP request timed out",
			Details: map[string]any{"port": port, "timeout_seconds": 120},
		}
	}

	// Connection refused: distinguish port_conflict vs stale_pid.
	if isConnectionRefused(err) {
		if pid > 0 && processAlive(pid) {
			return LocalSTTError{
				Reason:  "port_conflict",
				Message: "connection refused but server process is alive — port mismatch or another process took the port",
				Details: details,
			}
		}
		cleanStateFiles()
		return LocalSTTError{
			Reason:  "stale_pid",
			Message: "connection refused and server process is dead",
			Details: details,
		}
	}

	// Connection reset (process died during request) — likely OOM/crash.
	if isConnectionReset(err) {
		return LocalSTTError{
			Reason:  "oom",
			Message: "connection reset — server process may have been killed (OOM or crash)",
			Details: map[string]any{"port": port, "model": readModelFile()},
		}
	}

	// Fallback: server_down.
	return LocalSTTError{
		Reason:  "server_down",
		Message: fmt.Sprintf("local STT request failed: %v", err),
		Details: details,
	}
}

// classifySTTHTTPResponse inspects a non-200 HTTP response and classifies it
// into STT-specific failure modes. Mirrors classifyHTTPResponse from local.go
// but with STT-specific error patterns.
func classifySTTHTTPResponse(statusCode int, respBody []byte, port int) LocalSTTError {
	bodyStr := string(respBody)
	bodyPreview := bodyStr
	if len(bodyPreview) > 1024 {
		bodyPreview = bodyPreview[:1024]
	}

	lower := strings.ToLower(bodyStr)

	// HTTP 400/415/422 with unsupported format indicators.
	if statusCode == http.StatusBadRequest || statusCode == http.StatusUnsupportedMediaType || statusCode == 422 {
		if strings.Contains(lower, "unsupported") || strings.Contains(lower, "format") || strings.Contains(lower, "invalid") {
			return LocalSTTError{
				Reason:  "unsupported_format",
				Message: fmt.Sprintf("server rejected input (HTTP %d): unsupported audio format", statusCode),
				Details: map[string]any{"port": port, "http_status": statusCode, "body_preview": bodyPreview},
			}
		}
	}

	// HTTP 500 with model-related indicators.
	if statusCode == http.StatusInternalServerError {
		if strings.Contains(lower, "model") || strings.Contains(lower, "load") || strings.Contains(lower, "not found") {
			return LocalSTTError{
				Reason:  "model_error",
				Message: fmt.Sprintf("server model error (HTTP %d)", statusCode),
				Details: map[string]any{"port": port, "http_status": statusCode, "body_preview": bodyPreview},
			}
		}
		if strings.Contains(lower, "memory") || strings.Contains(lower, "oom") || strings.Contains(lower, "killed") {
			return LocalSTTError{
				Reason:  "oom",
				Message: fmt.Sprintf("server returned OOM/memory error (HTTP %d)", statusCode),
				Details: map[string]any{"port": port, "model": readModelFile(), "http_status": statusCode, "body_preview": bodyPreview},
			}
		}
	}

	// Default: server_down.
	return LocalSTTError{
		Reason:  "server_down",
		Message: fmt.Sprintf("server returned HTTP %d: %s", statusCode, bodyPreview),
		Details: map[string]any{"port": port, "http_status": statusCode, "body_preview": bodyPreview},
	}
}

// LocalSTT transcribes audio via HTTP POST (multipart form) to the running
// mlx-audio server's /v1/audio/transcriptions endpoint. The verbose JSON
// response is parsed into a structured transcript matching the existing
// extractTranscript format.
//
// All STT server errors are returned as LocalSTTError with a classified reason
// so the agent can take appropriate action. Non-server errors (path resolution,
// file I/O) are returned as their original error types.
func LocalSTT(inPath, model string) (SttResult, error) {
	// 1. Read port from state file.
	port := readPortFile()
	if port == 0 {
		return SttResult{}, LocalSTTError{
			Reason:  "server_down",
			Message: "server not running (no port file)",
			Details: map[string]any{"port": 0, "pid": 0},
		}
	}

	// 2. Read PID from state file and verify liveness.
	pid := readPIDFile()
	if pid == 0 {
		cleanStateFiles()
		return SttResult{}, LocalSTTError{
			Reason:  "server_down",
			Message: "server not running (no PID file)",
			Details: map[string]any{"port": port, "pid": 0},
		}
	}
	if !processAlive(pid) {
		cleanStateFiles()
		return SttResult{}, LocalSTTError{
			Reason:  "stale_pid",
			Message: "server process is dead (stale PID)",
			Details: map[string]any{"port": port, "pid": pid},
		}
	}

	// 3. Model resolution: default if empty (safety net for programmatic callers).
	if model == "" {
		model = DefaultSTTModel
	}

	// 4. Input file validation (resolve relative paths against workspace).
	inputPath, err := resolveInputPath(inPath)
	if err != nil {
		return SttResult{}, fmt.Errorf("Input file not found: %s", inPath)
	}

	// 5. Build multipart form body.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file field.
	fileField, err := writer.CreateFormFile("file", filepath.Base(inputPath))
	if err != nil {
		return SttResult{}, fmt.Errorf("failed to create multipart file field: %w", err)
	}
	f, err := os.Open(inputPath)
	if err != nil {
		return SttResult{}, fmt.Errorf("failed to open input file: %w", err)
	}
	if _, err := io.Copy(fileField, f); err != nil {
		f.Close()
		return SttResult{}, fmt.Errorf("failed to copy file to multipart: %w", err)
	}
	f.Close()

	// Add text fields.
	_ = writer.WriteField("model", model)
	_ = writer.WriteField("language", "zh")
	_ = writer.WriteField("response_format", "verbose_json")

	if err := writer.Close(); err != nil {
		return SttResult{}, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// 6. HTTP POST to /v1/audio/transcriptions.
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/audio/transcriptions", port)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return SttResult{}, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return SttResult{}, classifySTTHTTPError(err, port, pid)
	}
	defer resp.Body.Close()

	// 7. Check HTTP status.
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return SttResult{}, classifySTTHTTPResponse(resp.StatusCode, respBody, port)
	}

	// 8. Parse response JSON.
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return SttResult{}, fmt.Errorf("failed to read STT response body: %w", err)
	}
	var raw map[string]any
	if err := json.Unmarshal(respBytes, &raw); err != nil {
		return SttResult{}, fmt.Errorf("failed to parse STT response JSON: %w", err)
	}

	// Extract transcript using the same structure as extractTranscript.
	// Duration and confidence metadata are now extracted by extractTranscript itself.
	transcript := extractTranscript(raw)

	// 9. Write transcript artifact.
	now, err := clock.NowUTC()
	if err != nil {
		return SttResult{}, err
	}
	basename := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	transcriptPath, err := artifactPath("stt-local-"+basename, "json", now)
	if err != nil {
		return SttResult{}, err
	}
	content, err := jsonio.Dumps(transcript)
	if err != nil {
		return SttResult{}, err
	}
	if err := os.WriteFile(transcriptPath, []byte(content), 0o644); err != nil {
		return SttResult{}, err
	}

	workspace, err := paths.EnsureWorkspace()
	if err != nil {
		return SttResult{}, err
	}
	rel, err := relativeTo(workspace, transcriptPath)
	if err != nil {
		return SttResult{}, err
	}
	stat, err := os.Stat(transcriptPath)
	if err != nil {
		return SttResult{}, err
	}
	transcriptArtifact := envelope.Artifact{
		Path:    rel,
		MIME:    "application/json",
		Purpose: "transcript",
		Bytes:   intPtr(int(stat.Size())),
	}

	// 10. Return SttResult.
	data := map[string]any{
		"in":    inputPath,
		"model": model,
		"backend": map[string]any{
			"id":       "local",
			"features": []string{"stt"},
		},
		"transcript": transcript,
	}
	return SttResult{
		Data:      data,
		Artifacts: []envelope.Artifact{transcriptArtifact},
		Truncated: false,
		Limits:    map[string]any{},
	}, nil
}
