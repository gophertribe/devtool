package build

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// CanBuildLocally reports whether the effective target can be built on this
// host without spawning the build container.
func CanBuildLocally(effectiveOS, effectiveArch string) bool {
	if effectiveOS == runtime.GOOS && effectiveArch == runtime.GOARCH {
		return true
	}
	// amd64 cross targets (e.g. linux/amd64 -> windows/amd64) work with plain go.
	if effectiveArch == "amd64" {
		return true
	}
	// Same-OS arch cross (e.g. linux/amd64 -> linux/arm64) needs cross GCC in PATH.
	if effectiveOS == runtime.GOOS && hasCrossToolchain(effectiveArch) {
		return true
	}
	return false
}

func hasCrossToolchain(arch string) bool {
	var cc string
	switch arch {
	case "arm":
		cc = "arm-linux-gnueabihf-gcc"
	case "arm64":
		cc = "aarch64-linux-gnu-gcc"
	default:
		return false
	}
	_, err := exec.LookPath(cc)
	return err == nil
}

// BuildOutput returns the goreleaser-style binary path for a target and
// ensures the dist directory exists.
func BuildOutput(version, goos, goarch, packageName string) (string, error) {
	if err := os.MkdirAll("dist", 0o755); err != nil {
		return "", fmt.Errorf("could not create dist dir: %w", err)
	}
	name := fmt.Sprintf("dist/%s_%s_%s_%s", packageName, version, goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name, nil
}
