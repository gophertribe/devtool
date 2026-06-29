package build

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// CanBuildLocally reports whether the effective target can be built on this
// host with the Go compiler alone, i.e. without a C cross toolchain that is
// not present. It reflects only real go compiler constraints, not policy
// choices like build reproducibility (use the build container explicitly for
// that).
func CanBuildLocally(effectiveOS, effectiveArch string, cgoEnabled bool) bool {
	// Without cgo the Go toolchain cross-compiles to any GOOS/GOARCH on its
	// own, no external C toolchain required.
	if !cgoEnabled {
		return true
	}
	// With cgo, a native target builds with the host C toolchain.
	if effectiveOS == runtime.GOOS && effectiveArch == runtime.GOARCH {
		return true
	}
	// With cgo, cross-compilation needs a matching cross C toolchain in PATH.
	// Only same-OS arm/arm64 cross toolchains are detectable (and wired up by
	// GoBuild); anything else requires the build container.
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
