package build

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/gophertribe/devtool/execx"
)

// GoBuildOpts represents options for Go builds
type GoBuildOpts struct {
	EnableCgo     bool
	InjectVersion bool
	Version       string
	ConfigPackage string
	Tags          []string
	Arch          string
	OS            string
	GoPrivate     string
}

// GoBuild builds a Go application with the given options. It supports native
// builds and cross-compilation for any GOOS/GOARCH. A cross C toolchain is only
// required (and selected) when cgo is enabled and the target arch differs from
// the host arch.
func GoBuild(output, source string, opts GoBuildOpts) error {
	ldflags, err := buildLdflags(opts)
	if err != nil {
		return err
	}

	args := []string{"build", "-o", output, fmt.Sprintf("-ldflags=%s", ldflags)}
	if len(opts.Tags) > 0 {
		args = append(args, "-tags", strings.Join(opts.Tags, " "))
	}
	args = append(args, "-v", source)

	goos := runtime.GOOS
	if opts.OS != "" {
		goos = opts.OS
	}
	goarch := runtime.GOARCH
	if opts.Arch != "" {
		goarch = opts.Arch
	}

	cgoFlag := "0"
	if opts.EnableCgo {
		cgoFlag = "1"
	}

	env := map[string]string{
		"GOOS":        goos,
		"GOARCH":      goarch,
		"CGO_ENABLED": cgoFlag,
	}
	// Only override GOPRIVATE when explicitly provided, so we never clobber a
	// value inherited from the environment.
	if opts.GoPrivate != "" {
		env["GOPRIVATE"] = opts.GoPrivate
	}

	// A cross C toolchain is only needed when cgo is enabled and the target
	// architecture differs from the host.
	if opts.EnableCgo && goarch != runtime.GOARCH {
		if err := applyCrossToolchain(env, goarch); err != nil {
			return err
		}
	}

	if err := execx.RunWithV(env, "go", args...); err != nil {
		return fmt.Errorf("go build error: %w", err)
	}
	return nil
}

func applyCrossToolchain(env map[string]string, goarch string) error {
	switch goarch {
	case "arm":
		env["GOARM"] = "7"
		env["CC"] = "arm-linux-gnueabihf-gcc"
		env["CXX"] = "arm-linux-gnueabihf-g++"
		env["CGO_CFLAGS"] = "-march=armv7-a+fp"
		env["CGO_CXXFLAGS"] = "-march=armv7-a+fp"
	case "arm64":
		env["CC"] = "aarch64-linux-gnu-gcc"
		env["CXX"] = "aarch64-linux-gnu-g++"
		env["CGO_CFLAGS"] = "-march=armv8-a"
		env["CGO_CXXFLAGS"] = "-march=armv8-a"
	default:
		return fmt.Errorf("cgo cross-compilation is not supported for arch %q; disable cgo or add a toolchain", goarch)
	}
	return nil
}

func buildLdflags(opts GoBuildOpts) (string, error) {
	var flags strings.Builder
	flags.WriteString("-s -w")
	if !opts.InjectVersion {
		return flags.String(), nil
	}

	pwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not determine current path: %w", err)
	}
	repo, err := git.PlainOpen(pwd)
	if err != nil {
		return "", fmt.Errorf("could not open git repo: %w", err)
	}
	ref, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("could not establish current commit: %w", err)
	}
	fmt.Fprintf(&flags, " -X %s.AppVersion=%s", opts.ConfigPackage, opts.Version)
	fmt.Fprintf(&flags, " -X %s.GitCommit=%s", opts.ConfigPackage, ref.Hash().String()[:6])
	fmt.Fprintf(&flags, " -X %s.GitBranch=%s", opts.ConfigPackage, ref.Name().Short())
	fmt.Fprintf(&flags, " -X %s.BuildTime=%s", opts.ConfigPackage, time.Now().Format(time.RFC3339))
	fmt.Fprintf(&flags, " -X %s.Arch=x64", opts.ConfigPackage)
	return flags.String(), nil
}
