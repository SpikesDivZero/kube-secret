package cmd

import (
	"github.com/spf13/cobra"
)

var decodeCommand = &cobra.Command{
	Use:   "decode",
	Short: "Decode a kube secret file",
	Run:   runDecodeCommand,
	Args:  cobra.ExactArgs(1), // [filename.yml]
}

func runDecodeCommand(cmd *cobra.Command, args []string) {
	f := args[0]

	err := secretReadMungeWrite(f, f, "decode")
	if err != nil {
		errorExit(err)
	}
}

func init() {
	rootCmd.AddCommand(decodeCommand)
}
