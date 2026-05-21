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
		fmt.Fprintf(&flags, " -X %s.AppVersion=%s", opts.ConfigPackage, opts.Version)
		fmt.Fprintf(&flags, " -X %s.GitCommit=%s", opts.ConfigPackage, ref.Hash().String()[:6])
		fmt.Fprintf(&flags, " -X %s.GitBranch=%s", opts.ConfigPackage, ref.Name().Short())
		fmt.Fprintf(&flags, " -X %s.BuildTime=%s", opts.ConfigPackage, time.Now().Format(time.RFC3339))
		fmt.Fprintf(&flags, " -X %s.Arch=x64", opts.ConfigPackage)
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
		err := execx.RunWithV(map[string]string{
			"CGO_ENABLED": cgoFlag,
			"GOPRIVATE":   opts.GoPrivate,
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
		err := execx.RunWithV(map[string]string{
			"GOOS":         goos,
			"GOARCH":       goarch,
			"GOARM":        "7",
			"CGO_ENABLED":  cgoFlag,
			"GOPRIVATE":    "github.com/gophertribe,github.com/mklimuk,github.com/satsysoft",
			"CC":           "arm-linux-gnueabihf-gcc",
			"CXX":          "arm-linux-gnueabihf-g++",
			"AR":           "arm-linux-gnueabihf-ar",
			"CGO_CFLAGS":   "-march=armv7-a -mfpu=vfpv3-d16 -mfloat-abi=hard",
			"CGO_CXXFLAGS": "-march=armv7-a -mfpu=vfpv3-d16 -mfloat-abi=hard",
		}, "go", args...)
		if err != nil {
			return fmt.Errorf("go build error: %w", err)
		}
		return nil
	case "arm64":
		err := execx.RunWithV(map[string]string{
			"GOOS":         goos,
			"GOARCH":       goarch,
			"CGO_ENABLED":  cgoFlag,
			"GOPRIVATE":    "github.com/gophertribe,github.com/mklimuk,github.com/satsysoft",
			"CC":           "aarch64-linux-gnu-gcc",
			"CXX":          "aarch64-linux-gnu-g++",
			"AR":           "aarch64-linux-gnu-ar",
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
