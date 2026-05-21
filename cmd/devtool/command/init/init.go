package initcmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	templateassets "github.com/gophertribe/devtool/template"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize Makefile and minimal cmd/dev cobra app",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := copyMakefile(force); err != nil {
				return err
			}
			if err := scaffoldDevCLI(force); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing files")
	return cmd
}

func copyMakefile(force bool) error {
	data, err := templateassets.ReadMakefile()
	if err != nil {
		return fmt.Errorf("read embedded Makefile: %w", err)
	}
	target := filepath.Join("Makefile")
	if !force {
		if _, err := os.Stat(target); err == nil {
			//nolint:staticcheck
			return errors.New("Makefile already exists; use --force to overwrite")
		}
	}
	return os.WriteFile(target, data, 0o644)
}

func scaffoldDevCLI(force bool) error {
	// cmd/dev/main.go and cmd/dev/cmd/root.go
	mainPath := filepath.Join("cmd", "dev", "main.go")
	rootPath := filepath.Join("cmd", "dev", "cmd", "root.go")

	if !force {
		if _, err := os.Stat(mainPath); err == nil {
			return errors.New("cmd/dev already exists; use --force to overwrite")
		}
	}

	if err := os.MkdirAll(filepath.Dir(rootPath), 0o755); err != nil {
		return err
	}

	mainSrc := []byte(
		"package main\n\n" +
			"import (\n\t\"log\"\n\t\"github.com/spf13/cobra\"\n)\n\n" +
			"func main() {\n\trootCmd := &cobra.Command{Use: \"dev\", Short: \"Development CLI\"}\n\tif err := rootCmd.Execute(); err != nil { log.Fatal(err) }\n}\n",
	)

	rootSrc := []byte(
		"package cmd\n\n" +
			"import \"github.com/spf13/cobra\"\n\n" +
			"func NewRoot() *cobra.Command {\n\treturn &cobra.Command{Use: \"dev\", Short: \"Development CLI\"}\n}\n",
	)

	if err := os.WriteFile(mainPath, mainSrc, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(rootPath, rootSrc, 0o644); err != nil {
		return err
	}
	return nil
}
