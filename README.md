# M1 YouTube Transcriber

Local macOS desktop app (Apple Silicon friendly) to:

1. Paste a YouTube URL
2. Choose optional start/end time range
3. Transcribe locally via `whisper.cpp`
4. Export transcript as `.txt`
5. Optionally add timestamps every 10s or 50s
6. Open transcript in TextEdit
7. Delete downloaded media after processing

## Prerequisites (M1 Mac)

```bash
brew install yt-dlp ffmpeg
brew install whisper-cpp
```

Download a Whisper model file (for example `ggml-base.en.bin`) and note its path.

## Run

```bash
go mod tidy
go run .
```

## Build App

```bash
go build -o YTTranscriber .
```

You can package this binary into a `.app` bundle later (or use a packaging tool).

## Notes

- The app expects `whisper-cli` to be available in PATH.
- Output is a `.txt` file for TextEdit compatibility.
