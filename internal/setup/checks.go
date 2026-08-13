package setup

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

const (
	defaultModelName = "ggml-base.en.bin"
	defaultModelURL  = "https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-base.en.bin"
)

func ValidatePrerequisites() []error {
	var errs []error

	requiredBins := []string{"yt-dlp", "ffmpeg", "whisper-cli"}
	for _, bin := range requiredBins {
		if _, err := exec.LookPath(bin); err != nil {
			errs = append(errs, fmt.Errorf("%s not found. Install with Homebrew", bin))
		}
	}
	return errs
}

func ResolveBestModelPath(logf func(string)) (string, error) {
	if path := AutoDetectModelPath(); path != "" {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	modelDir := filepath.Join(home, "Library", "Application Support", "YTTranscriber", "models")
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return "", fmt.Errorf("create model directory: %w", err)
	}
	dest := filepath.Join(modelDir, defaultModelName)
	logf("Downloading built-in model (first run only)...")
	if err := downloadFile(defaultModelURL, dest); err != nil {
		return "", fmt.Errorf("download default model: %w", err)
	}
	return dest, nil
}

func AutoDetectModelPath() string {
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, "Library", "Application Support", "YTTranscriber", "models", defaultModelName),
		filepath.Join(home, "models", "ggml-base.en.bin"), // best balance for M1 speed/accuracy
		filepath.Join(home, "models", "ggml-small.en.bin"),
		filepath.Join(home, "models", "ggml-tiny.en.bin"),
		filepath.Join(home, "Downloads", "ggml-base.en.bin"),
		filepath.Join(home, "Downloads", "ggml-small.en.bin"),
		filepath.Join(home, "Downloads", "ggml-tiny.en.bin"),
	}
	for _, c := range candidates {
		info, err := os.Stat(c)
		if err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

func downloadFile(downloadURL, path string) error {
	resp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("model server returned non-success status")
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
