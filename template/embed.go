package templateassets

import (
	"embed"
	"io/fs"
)

// FS embeds files from the template directory.
//
//go:embed Makefile
var FS embed.FS

// ReadMakefile returns the Makefile template bytes.
func ReadMakefile() ([]byte, error) {
	return fs.ReadFile(FS, "Makefile")
}
