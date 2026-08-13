package pipeline

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"yttranscriber/internal/macos"
	"yttranscriber/internal/transcript"
)

type Config struct {
	URL            string
	StartTime      string
	EndTime        string
	OutputDir      string
	OutputFilename string
	TimestampMode  transcript.TimestampMode
	DeleteMedia    bool
	ModelPath      string
}

func Run(ctx context.Context, cfg Config, logf func(string)) (string, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return "", errors.New("youtube URL is required")
	}
	if err := validateYouTubeURL(cfg.URL); err != nil {
		return "", err
	}
	if strings.TrimSpace(cfg.ModelPath) == "" {
		return "", errors.New("model path is required")
	}
	if err := validateTimes(cfg.StartTime, cfg.EndTime); err != nil {
		return "", err
	}

	workDir, err := os.MkdirTemp("", "yt-transcriber-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	mediaPath := filepath.Join(workDir, "audio.m4a")
	trimmedPath := filepath.Join(workDir, "clip.wav")
	jsonPath := filepath.Join(workDir, "whisper_output.json")
	txtPath := filepath.Join(cfg.OutputDir, cfg.OutputFilename)

	shouldCleanup := cfg.DeleteMedia
	defer func() {
		if shouldCleanup {
			_ = Cleanup(workDir, logf)
		}
	}()

	logf("Downloading audio with yt-dlp...")
	if err := runCommand(ctx, "yt-dlp", []string{
		"-f", "bestaudio/best",
		"-o", mediaPath,
		cfg.URL,
	}, logf); err != nil {
		return "", fmt.Errorf("yt-dlp failed: %w", err)
	}

	inputForWhisper := mediaPath
	if strings.TrimSpace(cfg.StartTime) != "" || strings.TrimSpace(cfg.EndTime) != "" {
		logf("Trimming selected range with ffmpeg...")
		args := []string{"-y", "-i", mediaPath}
		if strings.TrimSpace(cfg.StartTime) != "" {
			args = append(args, "-ss", normalizeTime(cfg.StartTime))
		}
		if strings.TrimSpace(cfg.EndTime) != "" {
			args = append(args, "-to", normalizeTime(cfg.EndTime))
		}
		args = append(args, "-ar", "16000", "-ac", "1", trimmedPath)
		if err := runCommand(ctx, "ffmpeg", args, logf); err != nil {
			return "", fmt.Errorf("ffmpeg trim failed: %w", err)
		}
		inputForWhisper = trimmedPath
	}

	logf("Transcribing locally with whisper.cpp...")
	if err := runCommand(ctx, "whisper-cli", []string{
		"-m", cfg.ModelPath,
		"-f", inputForWhisper,
		"--output-json",
		"--output-file", strings.TrimSuffix(jsonPath, ".json"),
		"-l", "auto",
	}, logf); err != nil {
		return "", fmt.Errorf("whisper transcription failed: %w", err)
	}

	segments, err := readWhisperJSON(jsonPath)
	if err != nil {
		return "", fmt.Errorf("parse whisper output: %w", err)
	}

	logf("Formatting transcript...")
	outText := transcript.Format(segments, cfg.TimestampMode)
	if err := os.WriteFile(txtPath, []byte(outText), 0600); err != nil {
		return "", fmt.Errorf("write transcript: %w", err)
	}

	logf("Opening transcript in TextEdit...")
	if err := macos.OpenInTextEdit(ctx, txtPath); err != nil {
		return "", fmt.Errorf("open transcript in TextEdit: %w", err)
	}
	return txtPath, nil
}

func runCommand(ctx context.Context, bin string, args []string, logf func(string)) error {
	cmd := exec.CommandContext(ctx, bin, args...)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if text != "" {
		logf(text)
	}
	return err
}

func validateTimes(start, end string) error {
	if strings.TrimSpace(start) == "" || strings.TrimSpace(end) == "" {
		return nil
	}
	st, err := parseDurationLike(start)
	if err != nil {
		return fmt.Errorf("invalid start time: %w", err)
	}
	et, err := parseDurationLike(end)
	if err != nil {
		return fmt.Errorf("invalid end time: %w", err)
	}
	if et <= st {
		return errors.New("end time must be greater than start time")
	}
	return nil
}

func validateYouTubeURL(rawURL string) error {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return errors.New("invalid URL format")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return errors.New("URL must start with http:// or https://")
	}
	host := strings.ToLower(strings.TrimPrefix(u.Hostname(), "www."))
	if host != "youtube.com" && host != "m.youtube.com" && host != "youtu.be" {
		return errors.New("only YouTube URLs are allowed")
	}
	return nil
}

func normalizeTime(raw string) string {
	raw = strings.TrimSpace(raw)
	if strings.Count(raw, ":") == 1 {
		return "00:" + raw
	}
	return raw
}

func parseDurationLike(value string) (time.Duration, error) {
	parts := strings.Split(normalizeTime(value), ":")
	if len(parts) != 3 {
		return 0, errors.New("use HH:MM:SS or MM:SS")
	}
	h, m, s := parts[0], parts[1], parts[2]
	d, err := time.ParseDuration(fmt.Sprintf("%sh%sm%ss", h, m, s))
	if err != nil {
		return 0, err
	}
	return d, nil
}

type whisperFile struct {
	Transcription []struct {
		Text      string `json:"text"`
		Timestamp struct {
			From string `json:"from"`
			To   string `json:"to"`
		} `json:"timestamps"`
		Offsets struct {
			From int64 `json:"from"`
			To   int64 `json:"to"`
		} `json:"offsets"`
	} `json:"transcription"`
}

func readWhisperJSON(path string) ([]transcript.Segment, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wf whisperFile
	if err := json.Unmarshal(b, &wf); err == nil && len(wf.Transcription) > 0 {
		segments := make([]transcript.Segment, 0, len(wf.Transcription))
		for _, item := range wf.Transcription {
			segments = append(segments, transcript.Segment{
				StartMS: item.Offsets.From,
				EndMS:   item.Offsets.To,
				Text:    strings.TrimSpace(item.Text),
			})
		}
		return segments, nil
	}

	return nil, errors.New("unsupported whisper output format")
}
