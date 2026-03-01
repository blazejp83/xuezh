package audio

import (
	"bytes"
	"fmt"
	"io"
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

// LocalTTS synthesises speech via HTTP POST to the running mlx-audio server.
// The raw WAV response is converted to the target format (derived from outPath
// extension, defaulting to "ogg") using ffmpeg.
func LocalTTS(text, voice, outPath, purpose string) (AudioResult, error) {
	// 1. Read port from state file.
	port := readPortFile()
	if port == 0 {
		return AudioResult{}, fmt.Errorf("server not running")
	}

	// 2. Read model from state file.
	model := readModelFile()

	// 3. Verify server process is alive.
	pid := readPIDFile()
	if pid == 0 || !processAlive(pid) {
		cleanStateFiles()
		return AudioResult{}, fmt.Errorf("server not running (stale PID)")
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
		return AudioResult{}, fmt.Errorf("local TTS request failed: %w", err)
	}
	defer resp.Body.Close()

	// 9. Check HTTP status.
	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return AudioResult{}, fmt.Errorf("local TTS server returned %d: %s", resp.StatusCode, string(respBody))
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
