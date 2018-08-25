package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kube-secret",
	Short: "Making kube secret files a bit more bearable",
	Long:  "I should put something useful here.", // TODO
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprint(os.Stderr, err)
		fmt.Fprint(os.Stderr, "\n")
		os.Exit(1)
	}
}

var debug bool

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debugging output")
}
