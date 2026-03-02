package audio

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

// defaultSTTModel is the default Whisper model for local STT via mlx-audio.
// Phase 6 will make this configurable.
const defaultSTTModel = "mlx-community/whisper-large-v3-turbo"

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
