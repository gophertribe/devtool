package build

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/magefile/mage/sh"
)

// GoBuild builds a Go application with the given options
func GoBuild(output, source string, opts GoBuildOpts) error {
	var flags strings.Builder
	flags.WriteString("-s -w")
	if opts.InjectVersion {
		pwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not determine current path: %w", err)
		}
		repo, err := git.PlainOpen(pwd)
		if err != nil {
			return fmt.Errorf("could not open git repo: %w", err)
		}
		ref, err := repo.Head()
		if err != nil {
			return fmt.Errorf("could not establish current commit: %w", err)
		}
		flags.WriteString(fmt.Sprintf(" -X %s.AppVersion=%s", opts.ConfigPackage, opts.Version))
		flags.WriteString(fmt.Sprintf(" -X %s.GitCommit=%s", opts.ConfigPackage, ref.Hash().String()[:6]))
		flags.WriteString(fmt.Sprintf(" -X %s.GitBranch=%s", opts.ConfigPackage, ref.Name().Short()))
		flags.WriteString(fmt.Sprintf(" -X %s.BuildTime=%s", opts.ConfigPackage, time.Now().Format(time.RFC3339)))
		flags.WriteString(fmt.Sprintf(" -X %s.Arch=x64", opts.ConfigPackage))
	}

	var cgoFlag = "0"
	if opts.EnableCgo {
		cgoFlag = "1"
	}

	args := []string{"build", "-o", output, fmt.Sprintf(`-ldflags=%s`, flags.String())}
	if len(opts.Tags) > 0 {
		args = append(args, "-tags", strings.Join(opts.Tags, " "))
	}
	args = append(args, "-v", source)

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	// if we do not cross compile we run a simple build
	if (opts.OS == "" || opts.OS == goos) && (opts.Arch == "" || opts.Arch == goarch) {
		err := sh.RunWithV(map[string]string{
			"CGO_ENABLED": cgoFlag,
			"GOPRIVATE":   "github.com/gophertribe,github.com/mklimuk,github.com/satsysoft",
		}, "go", args...)
		if err != nil {
			return fmt.Errorf("go build error: %w", err)
		}
		return nil
	}

	if opts.Arch != "" {
		goarch = opts.Arch
	}
	if opts.OS != "" {
		goos = opts.OS
	}

	switch goarch {
	case "arm":
		err := sh.RunWithV(map[string]string{
			"GOOS":         goos,
			"GOARCH":       goarch,
			"GOARM":        "7",
			"CGO_ENABLED":  cgoFlag,
			"GOPRIVATE":    "github.com/gophertribe,github.com/mklimuk,github.com/satsysoft",
			"CC":           "arm-linux-gnueabihf-gcc",
			"CXX":          "arm-linux-gnueabihf-g++",
			"CGO_CFLAGS":   "-march=armv7-a+fp",
			"CGO_CXXFLAGS": "-march=armv7-a+fp",
		}, "go", args...)
		if err != nil {
			return fmt.Errorf("go build error: %w", err)
		}
		return nil
	case "arm64":
		err := sh.RunWithV(map[string]string{
			"GOOS":         goos,
			"GOARCH":       goarch,
			"CGO_ENABLED":  cgoFlag,
			"GOPRIVATE":    "github.com/gophertribe,github.com/mklimuk,github.com/satsysoft",
			"CC":           "aarch64-linux-gnu-gcc",
			"CXX":          "aarch64-linux-gnu-g++",
			"CGO_CFLAGS":   "-march=armv8-a",
			"CGO_CXXFLAGS": "-march=armv8-a",
		}, "go", args...)
		if err != nil {
			return fmt.Errorf("go build error: %w", err)
		}
		return nil
	}

	return fmt.Errorf("unsupported architecture: %s", goarch)
}
