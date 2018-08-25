package cmd

import (
	"fmt"
	"os"

	"github.com/spikesdivzero/kube-secret/pkg"
)

func errorExit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func readSecretFile(f string) (pkg.KubeSecretMunger, error) {
	ksm := pkg.NewKubeSecretMunger()
	ksm.SetDebug(debug) // from rootCmd

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
