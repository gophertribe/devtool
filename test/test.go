package test

import (
	"github.com/magefile/mage/sh"
)

// Test runs unit tests
func Test() error {
	return sh.RunV("go", "run", "gotest.tools/gotestsum@v1.12.0", "--no-summary=skipped", "--junitfile", "./coverage.xml", "--format", "short", "./...")
}

// Lint runs the linter
func Lint() error {
	return sh.RunV("go", "run", "github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.0", "run", "--timeout", "5m", "--skip-dirs", "dist", "./...")
}

// Integ runs integration tests
func Integ() error {
	env := map[string]string{
		"GOPRIVATE":                "github.com/gophertribe,github.com/mklimuk",
		"TEST_INTEGRATION_ENABLED": "1",
	}
	return sh.RunWithV(env, "go", "run", "gotest.tools/gotestsum@v1.12.0", "--no-summary=skipped", "--junitfile", "./coverage.xml", "--format", "short", "./...")
}
