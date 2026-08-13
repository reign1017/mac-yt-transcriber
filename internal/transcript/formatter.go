package transcript

import (
	"fmt"
	"strings"
)

type TimestampMode string

const (
	TimestampNone TimestampMode = "none"
	Timestamp10s  TimestampMode = "10s"
	Timestamp50s  TimestampMode = "50s"
)

type Segment struct {
	StartMS int64
	EndMS   int64
	Text    string
}

func Format(segments []Segment, mode TimestampMode) string {
	if len(segments) == 0 {
		return ""
	}

	var out strings.Builder
	switch mode {
	case Timestamp10s:
		return formatByInterval(segments, 10_000)
	case Timestamp50s:
		return formatByInterval(segments, 50_000)
	default:
		for _, s := range segments {
			t := strings.TrimSpace(s.Text)
			if t == "" {
				continue
			}
			out.WriteString(t)
			out.WriteString(" ")
		}
		return strings.TrimSpace(out.String())
	}
}

func formatByInterval(segments []Segment, intervalMs int64) string {
	var out strings.Builder
	lastBucket := int64(-1)

	for _, s := range segments {
		if strings.TrimSpace(s.Text) == "" {
			continue
		}
		bucket := s.StartMS / intervalMs
		if bucket != lastBucket {
			if out.Len() > 0 {
				out.WriteString("\n")
			}
			out.WriteString(fmt.Sprintf("[%s] ", formatTimeMs(bucket*intervalMs)))
			lastBucket = bucket
		}
		out.WriteString(strings.TrimSpace(s.Text))
		out.WriteString(" ")
	}
	return strings.TrimSpace(out.String())
}

func formatTimeMs(ms int64) string {
	totalSeconds := ms / 1000
	h := totalSeconds / 3600
	m := (totalSeconds % 3600) / 60
	s := totalSeconds % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}
