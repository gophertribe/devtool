package main

import (
	"fmt"

	initcmd "github.com/gophertribe/devtool/cmd/devtool/command/init"
	"github.com/spf13/cobra"
)

func main() {
	root := &cobra.Command{
		Use:   "devtool",
		Short: "devtool scaffolds a dev CLI and Makefile",
	}
	root.AddCommand(initcmd.New())

	if err := root.Execute(); err != nil {
		fmt.Println(err)
	}
}
