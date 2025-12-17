package test

import (
	"github.com/magefile/mage/sh"
)

// Test runs unit tests
func Test() error {
	env := map[string]string{
		"GOPRIVATE": "github.com/gophertribe,github.com/mklimuk,github.com/satsysoft",
	}
	return sh.RunWithV(env, "go", "run", "gotest.tools/gotestsum@v1.13.0", "--no-summary=skipped", "--junitfile", "./coverage.xml", "--format", "short", "./...")
}

// Lint runs the linter
func Lint() error {
	env := map[string]string{
		"GOPRIVATE": "github.com/gophertribe,github.com/mklimuk,github.com/satsysoft",
	}
	return sh.RunWithV(env, "go", "run", "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2", "run", "--timeout", "5m", "./...")
}

// Integ runs integration tests
func Integ() error {
	env := map[string]string{
		"GOPRIVATE":                "github.com/gophertribe,github.com/mklimuk,github.com/satsysoft",
		"TEST_INTEGRATION_ENABLED": "1",
	}
	return sh.RunWithV(env, "go", "run", "gotest.tools/gotestsum@v1.13.0", "--no-summary=skipped", "--junitfile", "./coverage.xml", "--format", "short", "./...")
}
