package execx

import "testing"

func TestSortedEnvPairs(t *testing.T) {
	got := sortedEnvPairs(map[string]string{
		"GOPRIVATE": "github.com/acme/private",
		"GOOS":      "linux",
		"EMPTY":     "",
	})

	want := []string{
		"EMPTY=''",
		"GOOS=linux",
		"GOPRIVATE=github.com/acme/private",
	}

	if len(got) != len(want) {
		t.Fatalf("len(sortedEnvPairs()) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sortedEnvPairs()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestQuoteShellArg(t *testing.T) {
	tests := map[string]string{
		"plain":                "plain",
		"hello world":          "'hello world'",
		"":                     "''",
		"can't-stop":           "'can'\\''t-stop'",
		`contains"double"`:     `'contains"double"'`,
		`path/with/slashes`:    "path/with/slashes",
		"embedded\twhitespace": "'embedded\twhitespace'",
	}

	for input, want := range tests {
		if got := quoteShellArg(input); got != want {
			t.Fatalf("quoteShellArg(%q) = %q, want %q", input, got, want)
		}
	}
}
