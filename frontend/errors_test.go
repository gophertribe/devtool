package frontend

import (
	"strings"
	"testing"

	"github.com/evanw/esbuild/pkg/api"
)

func TestFormatBuildErrors(t *testing.T) {
	t.Parallel()

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		if err := FormatBuildErrors(nil); err != nil {
			t.Fatalf("FormatBuildErrors(nil) = %v, want nil", err)
		}
	})

	t.Run("nil location", func(t *testing.T) {
		t.Parallel()
		err := FormatBuildErrors([]api.Message{
			{Text: "could not resolve entry point"},
		})
		want := "build failed: could not resolve entry point"
		if got := err.Error(); got != want {
			t.Fatalf("err.Error() = %q, want %q", got, want)
		}
	})

	t.Run("located message", func(t *testing.T) {
		t.Parallel()
		err := FormatBuildErrors([]api.Message{
			{
				Text: "Unexpected token",
				Location: &api.Location{
					File:       "src/main.js",
					Line:       12,
					Column:     4,
					LineText:   "foo bar",
					Suggestion: "Did you mean 'baz'?",
				},
			},
		})
		want := strings.Join([]string{
			"build failed: src/main.js:12:4: Unexpected token",
			"  foo bar",
			"  Did you mean 'baz'?",
		}, "\n")
		if got := err.Error(); got != want {
			t.Fatalf("err.Error() = %q, want %q", got, want)
		}
	})
}
