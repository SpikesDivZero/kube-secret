package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var encodeCommand = &cobra.Command{
	Use:   "encode",
	Short: "Encode a kube secret file",
	Long:  "I should put something useful here.", // TODO
	Run:   runEncodeCommand,
	Args:  cobra.ExactArgs(1), // [filename.yml]
}

func runEncodeCommand(cmd *cobra.Command, args []string) {
	f := args[0]

	err := secretReadMungeWrite(f, f, "encode")
	if err != nil {
		errorExit(err)
	}

	fmt.Fprintln(os.Stderr, "All done!")
}

func init() {
	rootCmd.AddCommand(encodeCommand)
}
