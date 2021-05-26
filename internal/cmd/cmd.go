package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kube-secret",
	Short: "Making kube secret files a bit more bearable",
}

var yamlFileExtensions = map[string]bool{
	".yml":  true,
	".yaml": true,
}

// Only intended as a crude convenience check; misses a lot of edge cases.
func isYamlFile(name string) bool {
	if !yamlFileExtensions[filepath.Ext(name)] {
		return false
	}

	stat, err := os.Stat(name)
	if err != nil {
		return false
	}
	return !stat.IsDir()
}

func Execute() {
	// For compatibility with "kubectl secret edit", if the first parameter is a filename, then inject the edit
	// command to make it work.
	if len(os.Args) > 1 && isYamlFile(os.Args[1]) {
		os.Args = append([]string{os.Args[0], "edit"}, os.Args[1:]...)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var debug bool

func init() {
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debugging output")
}
