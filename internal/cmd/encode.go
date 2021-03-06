package cmd

import (
	"github.com/spf13/cobra"
)

var encodeCommand = &cobra.Command{
	Use:   "encode",
	Short: "Encode a kube secret file",
	Run:   runEncodeCommand,
	Args:  cobra.ExactArgs(1), // [filename.yml]
}

func runEncodeCommand(cmd *cobra.Command, args []string) {
	f := args[0]

	err := secretReadMungeWrite(f, f, "encode")
	if err != nil {
		errorExit(err)
	}
}

func init() {
	rootCmd.AddCommand(encodeCommand)
}
