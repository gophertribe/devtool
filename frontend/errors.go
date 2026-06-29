package frontend

import (
	"fmt"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

func formatBuildErrorMessage(msg api.Message) string {
	if msg.Location == nil {
		return msg.Text
	}
	line := fmt.Sprintf("%s:%d:%d: %s",
		msg.Location.File, msg.Location.Line, msg.Location.Column, msg.Text)
	if msg.Location.LineText != "" {
		line += "\n  " + msg.Location.LineText
	}
	if msg.Location.Suggestion != "" {
		line += "\n  " + msg.Location.Suggestion
	}
	return line
}

func FormatBuildErrors(messages []api.Message) error {
	if len(messages) == 0 {
		return nil
	}
	lines := make([]string, 0, len(messages))
	for _, msg := range messages {
		lines = append(lines, formatBuildErrorMessage(msg))
	}
	return fmt.Errorf("%w: %s", ErrBuildFailed, strings.Join(lines, "\n"))
}
