package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"yttranscriber/internal/pipeline"
	"yttranscriber/internal/setup"
	"yttranscriber/internal/transcript"
)

func main() {
	a := app.NewWithID("local.yt.transcriber.m1")
	iconRes := loadAppIcon()
	if iconRes != nil {
		a.SetIcon(iconRes)
	}

	w := a.NewWindow("M1 YouTube Transcriber")
	w.Resize(fyne.NewSize(900, 720))

	home, _ := os.UserHomeDir()
	defaultOutDir := home
	defaultName := "transcript.txt"

	urlEntry := widget.NewEntry()
	urlEntry.SetPlaceHolder("https://www.youtube.com/watch?v=...")

	startEntry := widget.NewEntry()
	startEntry.SetPlaceHolder("Optional - HH:MM:SS or MM:SS")

	endEntry := widget.NewEntry()
	endEntry.SetPlaceHolder("Optional - HH:MM:SS or MM:SS")

	urlHint := widget.NewLabel("Paste YouTube link. Start time auto-detects from the link when available.")
	urlHint.Wrapping = fyne.TextWrapWord

	outputDirLabel := widget.NewLabel(defaultOutDir)
	outputDirButton := widget.NewButton("Choose Output Folder", func() {
		fd := dialog.NewFolderOpen(func(u fyne.ListableURI, err error) {
			if err != nil || u == nil {
				return
			}
			defaultOutDir = u.Path()
			outputDirLabel.SetText(defaultOutDir)
		}, w)
		fd.Show()
	})

	fileNameEntry := widget.NewEntry()
	fileNameEntry.SetText(defaultName)

	timestampSelect := widget.NewSelect([]string{
		"Normal transcript",
		"10 sec",
		"50 sec",
	}, nil)
	timestampSelect.SetSelected("Normal transcript")

	deleteMediaCheck := widget.NewCheck("Delete downloaded media after transcript", nil)
	deleteMediaCheck.SetChecked(true)

	status := widget.NewMultiLineEntry()
	status.Wrapping = fyne.TextWrapWord
	status.SetMinRowsVisible(9)
	status.Disable()

	progress := widget.NewProgressBarInfinite()
	progress.Hide()

	runButton := widget.NewButton("Download Transcript (Local)", nil)
	runButton.OnTapped = func() {
		runButton.Disable()
		progress.Show()

		selectedMode := transcript.TimestampNone
		if timestampSelect.Selected == "10 sec" {
			selectedMode = transcript.Timestamp10s
		}
		if timestampSelect.Selected == "50 sec" {
			selectedMode = transcript.Timestamp50s
		}

		cfg := pipeline.Config{
			URL:            strings.TrimSpace(urlEntry.Text),
			StartTime:      strings.TrimSpace(startEntry.Text),
			EndTime:        strings.TrimSpace(endEntry.Text),
			OutputDir:      defaultOutDir,
			OutputFilename: sanitizeOutputName(fileNameEntry.Text),
			TimestampMode:  selectedMode,
			DeleteMedia:    deleteMediaCheck.Checked,
		}

		appendStatus(status, "Running dependency checks...")
		if errs := setup.ValidatePrerequisites(); len(errs) > 0 {
			appendStatus(status, "Missing prerequisites:")
			for _, e := range errs {
				appendStatus(status, "- "+e.Error())
			}
			progress.Hide()
			runButton.Enable()
			return
		}
		modelPath, err := setup.ResolveBestModelPath(func(msg string) {
			appendStatus(status, msg)
		})
		if err != nil {
			progress.Hide()
			runButton.Enable()
			appendStatus(status, "Failed: "+err.Error())
			dialog.ShowError(err, w)
			return
		}
		cfg.ModelPath = modelPath

		logger := func(msg string) {
			appendStatus(status, msg)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Hour)
		defer cancel()

		out, err := pipeline.Run(ctx, cfg, logger)
		progress.Hide()
		runButton.Enable()
		if err != nil {
			appendStatus(status, "Failed: "+err.Error())
			dialog.ShowError(err, w)
			return
		}
		appendStatus(status, fmt.Sprintf("Complete. Transcript saved at %s", out))
		dialog.ShowInformation("Success", "Transcript created and opened in TextEdit.", w)
	}

	urlEntry.OnChanged = func(s string) {
		start, ok := extractStartTimeFromYouTubeURL(s)
		if ok && strings.TrimSpace(startEntry.Text) == "" {
			startEntry.SetText(start)
		}
	}

	logo := canvas.NewImageFromResource(iconRes)
	logo.FillMode = canvas.ImageFillContain
	logo.SetMinSize(fyne.NewSize(84, 84))
	if iconRes == nil {
		logo.Hide()
	}

	headerTitle := widget.NewLabel("M1 YouTube Transcriber")
	headerSubtitle := widget.NewLabel("Parabolic-style flow: paste URL, pick range, choose timestamp style, download transcript.")
	headerSubtitle.Wrapping = fyne.TextWrapWord
	header := container.NewHBox(
		logo,
		container.NewVBox(headerTitle, headerSubtitle),
	)

	mainCard := widget.NewCard(
		"New Transcript",
		"",
		container.NewVBox(
			widget.NewLabel("YouTube URL"),
			urlEntry,
			urlHint,
			widget.NewSeparator(),
			widget.NewLabel("Time Range"),
			container.NewGridWithColumns(2, startEntry, endEntry),
			widget.NewSeparator(),
			widget.NewLabel("Output"),
			container.NewHBox(outputDirLabel, layout.NewSpacer(), outputDirButton),
			fileNameEntry,
			widget.NewSeparator(),
			widget.NewLabel("Timestamp Style"),
			timestampSelect,
			deleteMediaCheck,
			runButton,
			progress,
		),
	)

	statusCard := widget.NewCard("Activity", "", status)

	form := container.NewVBox(header, mainCard, statusCard)

	w.SetContent(container.NewPadded(form))
	w.ShowAndRun()
}

func extractStartTimeFromYouTubeURL(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", false
	}
	q := u.Query()
	for _, key := range []string{"t", "start"} {
		val := strings.TrimSpace(q.Get(key))
		if val == "" {
			continue
		}
		secs, ok := parseYouTubeTimeToSeconds(val)
		if !ok {
			continue
		}
		return formatSeconds(secs), true
	}
	return "", false
}

func parseYouTubeTimeToSeconds(raw string) (int, bool) {
	raw = strings.TrimSpace(strings.TrimSuffix(strings.ToLower(raw), "s"))
	if raw == "" {
		return 0, false
	}
	if strings.Contains(raw, ":") {
		parts := strings.Split(raw, ":")
		if len(parts) == 2 {
			m, errM := strconv.Atoi(parts[0])
			s, errS := strconv.Atoi(parts[1])
			if errM != nil || errS != nil {
				return 0, false
			}
			return (m * 60) + s, true
		}
		if len(parts) == 3 {
			h, errH := strconv.Atoi(parts[0])
			m, errM := strconv.Atoi(parts[1])
			s, errS := strconv.Atoi(parts[2])
			if errH != nil || errM != nil || errS != nil {
				return 0, false
			}
			return (h * 3600) + (m * 60) + s, true
		}
		return 0, false
	}
	secs, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return secs, true
}

func formatSeconds(seconds int) string {
	if seconds < 0 {
		seconds = 0
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func loadAppIcon() fyne.Resource {
	b, err := os.ReadFile("assets/logo.svg")
	if err != nil {
		return nil
	}
	return fyne.NewStaticResource("logo.svg", b)
}

func appendStatus(entry *widget.Entry, msg string) {
	if entry.Text == "" {
		entry.SetText(msg)
		return
	}
	entry.SetText(entry.Text + "\n" + msg)
}

func sanitizeOutputName(name string) string {
	base := strings.TrimSpace(name)
	if base == "" {
		return "transcript.txt"
	}
	base = filepath.Base(base)
	if !strings.HasSuffix(strings.ToLower(base), ".txt") {
		base += ".txt"
	}
	return base
}
