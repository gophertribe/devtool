package console

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

// Available ANSI colors
var (
	Yellow = color.New(color.FgYellow).SprintFunc()
	Red    = color.New(color.FgRed).SprintFunc()
	Green  = color.New(color.FgGreen).SprintFunc()
	White  = color.New(color.FgHiWhite).SprintFunc()
	Bold   = color.New(color.Bold).SprintFunc()
	Cyan   = color.New(color.FgCyan).SprintFunc()
	Dim    = color.New(color.Faint).SprintFunc()
)

// Error writes msg to w using the standard error color.
func Error(w io.Writer, msg string) {
	_, _ = fmt.Fprintln(w, Red(msg))
}
