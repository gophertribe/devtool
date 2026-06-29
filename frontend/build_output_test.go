package frontend

import (
	"bytes"
	"strings"
	"testing"

	"github.com/evanw/esbuild/pkg/api"
)

func TestCleanBuildDirDefault(t *testing.T) {
	t.Parallel()
	if !cleanBuildDir(BuildOptions{}) {
		t.Fatal("expected CleanBuildDir to default to true")
	}
}

func TestCleanBuildDirExplicit(t *testing.T) {
	t.Parallel()

	falseVal := false
	if cleanBuildDir(BuildOptions{CleanBuildDir: &falseVal}) {
		t.Fatal("expected CleanBuildDir=false to disable cleaning")
	}

	trueVal := true
	if !cleanBuildDir(BuildOptions{CleanBuildDir: &trueVal}) {
		t.Fatal("expected CleanBuildDir=true to enable cleaning")
	}
}

func TestShouldPrintWarnings(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts BuildOptions
		want bool
	}{
		{name: "dev mode", opts: BuildOptions{Mode: "dev"}, want: true},
		{name: "prod mode", opts: BuildOptions{Mode: "prod"}, want: false},
		{name: "prod verbose", opts: BuildOptions{Mode: "prod", Verbose: true}, want: true},
		{name: "dev verbose", opts: BuildOptions{Mode: "dev", Verbose: true}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := shouldPrintWarnings(tt.opts); got != tt.want {
				t.Fatalf("shouldPrintWarnings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrintBuildWarnings(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	opts := BuildOptions{
		Mode:   "dev",
		ErrOut: &out,
	}
	printBuildWarnings(opts, []api.Message{
		{Text: "unused import"},
		{
			Text: "deprecated API",
			Location: &api.Location{
				File:     "src/app.js",
				Line:     3,
				Column:   5,
				LineText: "  oldCall()",
			},
		},
	})

	got := out.String()
	if !strings.Contains(got, "unused import") {
		t.Fatalf("expected warning text in output, got %q", got)
	}
	if !strings.Contains(got, "src/app.js:3:5") {
		t.Fatalf("expected located warning in output, got %q", got)
	}
	if !strings.Contains(got, "oldCall()") {
		t.Fatalf("expected line text in output, got %q", got)
	}
}

func TestPrintBuildWarningsSkippedInProd(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	opts := BuildOptions{
		Mode:   "prod",
		ErrOut: &out,
	}
	printBuildWarnings(opts, []api.Message{{Text: "unused import"}})
	if out.Len() != 0 {
		t.Fatalf("expected no warnings in prod mode, got %q", out.String())
	}
}
