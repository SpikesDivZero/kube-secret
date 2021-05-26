package cmd

import (
	"fmt"
	"os"

	"github.com/spikesdivzero/kube-secret/internal"
)

func errorExit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func readSecretFile(f string) (internal.KubeSecretMunger, error) {
	ksm := internal.NewKubeSecretMunger()

	inFile, err := os.Open(f)
	if err != nil {
		return nil, fmt.Errorf("Error opening %q for reading: %s", f, err)
	}
	defer inFile.Close()

	err = ksm.ReadFrom(inFile)
	if err != nil {
		return nil, fmt.Errorf("Error reading secrets from %q: %s", f, err)
	}

	return ksm, nil
}

func writeSecretFile(f string, ksm internal.KubeSecretMunger) error {
	fh, err := os.OpenFile(f, os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("Error opening %q for writing: %s", f, err)
	}
	defer fh.Close()

	return ksm.WriteTo(fh)
}

// ReadFrom, munge, WriteTo, all wrapped up in a nice tidy utility.
//
// This is specifically written in such a way so as to allow us to read
// and write to the same file -- that is, we close the in file as soon
// as we're done reading, before we open the out file for writing (since our
// writes are done in Truncate mode)
func secretReadMungeWrite(inF, outF, opType string) error {
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

	return writeSecretFile(outF, ksm)
}

// Use KUBE_EDITOR if defined, falling back to EDITOR, then just defaulting to vi.
func whichEditor() string {
	if e, ok := os.LookupEnv("KUBE_EDITOR"); ok {
		return e
	}
	if e, ok := os.LookupEnv("EDITOR"); ok {
		return e
	}
	return "vi"
}
