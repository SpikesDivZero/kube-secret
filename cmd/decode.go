package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var decodeCommand = &cobra.Command{
	Use:   "decode",
	Short: "Decode a kube secret file",
	Long:  "I should put something useful here.", // TODO
	Run:   runDecodeCommand,
	Args:  cobra.ExactArgs(1), // [filename.yml]
}

func runDecodeCommand(cmd *cobra.Command, args []string) {
	f := args[0]

	err := secretReadMungeWrite(f, f, "decode")
	if err != nil {
		errorExit(err)
	}

	fmt.Fprintln(os.Stderr, "All done!")
}

func init() {
	rootCmd.AddCommand(decodeCommand)
}
