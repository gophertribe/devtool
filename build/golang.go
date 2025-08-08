package build

import (
	"fmt"
	"strings"

	"github.com/magefile/mage/sh"
)

// GoBuild builds a Go application with the given options
func GoBuild(output, source string, opts GoBuildOpts) error {
	args := []string{"build"}

	if opts.InjectVersion {
		args = append(args, "-ldflags", fmt.Sprintf("-X %s.Version=%s", opts.ConfigPackage, opts.Version))
	}

	if opts.EnableCgo {
		args = append(args, "-tags", "cgo")
	}

	if len(opts.Tags) > 0 {
		args = append(args, "-tags", strings.Join(opts.Tags, ","))
	}

	args = append(args, "-o", output, source)
	return sh.Run("go", args...)
}
