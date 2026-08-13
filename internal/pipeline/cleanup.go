package pipeline

import (
	"fmt"
	"os"
)

func Cleanup(tempPath string, logf func(string)) error {
	if tempPath == "" {
		return nil
	}
	if err := os.RemoveAll(tempPath); err != nil {
		return fmt.Errorf("cleanup temp media failed: %w", err)
	}
	logf("Temporary media deleted.")
	return nil
}
