package frontend

import (
	"fmt"
	"io"
	"os"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/gophertribe/devtool/console"
)

func cleanBuildDir(opts BuildOptions) bool {
	if opts.CleanBuildDir == nil {
		return true
	}
	return *opts.CleanBuildDir
}

func shouldPrintWarnings(opts BuildOptions) bool {
	if opts.Verbose {
		return true
	}
	return opts.Mode != "prod"
}

func buildOut(opts BuildOptions) io.Writer {
	if opts.ErrOut != nil {
		return opts.ErrOut
	}
	return os.Stderr
}

func prepareBuildDir(buildDir string, clean bool) error {
	if clean {
		if err := os.RemoveAll(buildDir); err != nil {
			return fmt.Errorf("could not clear build dir: %w", err)
		}
		return os.Mkdir(buildDir, 0755)
	}
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return fmt.Errorf("could not create build dir: %w", err)
	}
	return nil
}

func printBuildWarnings(opts BuildOptions, warnings []api.Message) {
	if !shouldPrintWarnings(opts) || len(warnings) == 0 {
		return
	}
	out := buildOut(opts)
	for _, warning := range warnings {
		printBuildMessage(out, warning, console.Yellow)
	}
}

func printBuildMessage(w io.Writer, msg api.Message, highlight func(...any) string) {
	if msg.Location == nil {
		_, _ = fmt.Fprintln(w, highlight(msg.Text))
		return
	}
	loc := fmt.Sprintf("%s:%d:%d", msg.Location.File, msg.Location.Line, msg.Location.Column)
	_, _ = fmt.Fprintf(w, "%s %s\n", highlight(loc), msg.Text)
	if msg.Location.LineText != "" {
		_, _ = fmt.Fprintf(w, "%s\n", msg.Location.LineText)
	}
	if msg.Location.Suggestion != "" {
		_, _ = fmt.Fprintf(w, "%s\n", msg.Location.Suggestion)
	}
}
