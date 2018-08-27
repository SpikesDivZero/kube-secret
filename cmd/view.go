package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var viewCommand = &cobra.Command{
	Use:   "view",
	Short: "view a kube secret file",
	Run:   runViewCommand,
	Args:  cobra.ExactArgs(1), // [filename.yml]
}

func runViewCommand(cmd *cobra.Command, args []string) {
	ksm, err := readSecretFile(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = ksm.DecodeSecrets()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding secrets from %q: %s\n", args[0], err)
		os.Exit(1)
	}

	err = ksm.WriteTo(os.Stdout)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error writing decoded file to STDOUT: %s\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(viewCommand)
}
