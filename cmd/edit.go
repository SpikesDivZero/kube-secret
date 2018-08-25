package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var editCommand = &cobra.Command{
	Use:   "edit",
	Short: "Edit a kube secret file",
	Long:  "I should put something useful here.", // TODO
	Run:   runEditCommand,
	Args:  cobra.ExactArgs(1), // [filename.yml]
}

// This isn't particularly secure in how we use this (race conditions and what
// not), but it's good enough. Caller is responsible for `defer os.Remove(name)`
func getTempFileName(dir, pattern string) (string, error) {
	fh, err := ioutil.TempFile("", "ksed*.yml")
	if err != nil {
		return "", err
	}

	name := fh.Name()
	fh.Close()
	return name, nil
}

// ReadFrom, munge, WriteTo, all wrapped up in a nice tidy utility.
func edit_readMungeWrite(inF, outF, opType string) error {
	ksm, err := readSecretFile(inF)
	if err != nil {
		return err
	}

	switch opType {
	case "encode":
		err = ksm.EncodeSecrets()
	case "decode":
		err = ksm.DecodeSecrets()
	default:
		panic("Unknown opType: " + opType)
	}
	if err != nil {
		return fmt.Errorf("Error %s secrets from %q: %s", opType, inF, err)
	}

	outFh, err := os.OpenFile(outF, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("Error opening %q for writing: %s", outF, err)
	}
	defer outFh.Close()

	return ksm.WriteTo(outFh)
}

func runEditor(ed, f string) error {
	cmd := exec.Command(ed, f)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Error invoking %q: %s", ed, err)
	}
	return err
}

func runEditCommand(cmd *cobra.Command, args []string) {
	inF := args[0]
	editor := whichEditor()

	tmpF, err := getTempFileName("", "ksed*.yml")
	if err != nil {
		errorExit(err)
	}
	defer os.Remove(tmpF)

	err = edit_readMungeWrite(inF, tmpF, "decode")
	if err != nil {
		errorExit(err)
	}

	err = runEditor(editor, tmpF)
	if err != nil {
		errorExit(err)
	}

	err = edit_readMungeWrite(inF, tmpF, "encode")
	if err != nil {
		errorExit(err)
	}

	fmt.Fprintln(os.Stderr, "All done!")
}

func init() {
	rootCmd.AddCommand(editCommand)
}
