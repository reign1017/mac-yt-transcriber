package macos

import (
	"context"
	"os/exec"
)

func OpenInTextEdit(ctx context.Context, path string) error {
	return exec.CommandContext(ctx, "open", "-a", "TextEdit", path).Run()
}
