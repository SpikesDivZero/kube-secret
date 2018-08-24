package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"gopkg.in/yaml.v2"
)

// Returns a pointer to the MapItem with the corresponding Key.
func findKey(ms yaml.MapSlice, key string) *yaml.MapItem {
	for idx, _ := range ms {
		kv := &ms[idx]
		if kv.Key == key {
			return kv
		}
	}
	return nil
}

func ensureIsSecret(data yaml.MapSlice) error {
	kv := findKey(data, "kind")
	if kv == nil {
		return fmt.Errorf("Yaml file does not have a `kind`")
	}

	kind, ok := kv.Value.(string)
	if !ok {
		return fmt.Errorf("Yaml file `kind` is %#v, expected string", kv.Value)
	}
	if kind != "Secret" {
		return fmt.Errorf("Yaml file `kind` is %q, expected 'Secret'", kind)
	}
	return nil
}

// A callback which munges a yaml.MapItem in-place.
type secretDataMunger func(kv *yaml.MapItem) error

func secretDataDecoder(kv *yaml.MapItem) error {
	secret, ok := kv.Value.(string)
	if !ok {
		return fmt.Errorf("Secret %q is %#v, expected string", kv.Value)
	}

	decoded, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return fmt.Errorf("Secret $q is %#v, failed to decode base64", kv.Value)
	}

	kv.Value = string(decoded)
	return nil
}

func secretDataEncoder(kv *yaml.MapItem) error {
	secret, ok := kv.Value.(string)
	if !ok {
		return fmt.Errorf("Secret %q is %#v, expected string", kv.Value)
	}

	kv.Value = base64.StdEncoding.EncodeToString([]byte(secret))
	return nil
}

// Applies a secretDataMunger to all values in data
func mapAcrossData(data yaml.MapSlice, fn secretDataMunger) error {
	for idx, _ := range data {
		if err := fn(&data[idx]); err != nil {
			return err
		}
	}

	return nil
}

// An interface to cover decodeSecretData and encodeSecretData
type yamlSecretFileMunger func(yaml.MapSlice) error

// Decodes the data in-place. Arg is the root yaml struct.
func decodeSecretData(data yaml.MapSlice) error {
	if err := ensureIsSecret(data); err != nil {
		return err
	}

	kv := findKey(data, "data")
	if kv == nil {
		return fmt.Errorf("Yaml file does not have a `data`?")
	}

	secretData, ok := kv.Value.(yaml.MapSlice)
	if !ok {
		return fmt.Errorf("Yaml file `data` is %#v, expected MapSlice", kv.Value)
	}

	return mapAcrossData(secretData, secretDataDecoder)
}

func encodeSecretData(data yaml.MapSlice) error {
	if err := ensureIsSecret(data); err != nil {
		return err
	}

	kv := findKey(data, "data")
	if kv == nil {
		return fmt.Errorf("Yaml file does not have a `data`?")
	}

	secretData, ok := kv.Value.(yaml.MapSlice)
	if !ok {
		return fmt.Errorf("Yaml file `data` is %#v, expected MapSlice", kv.Value)
	}

	return mapAcrossData(secretData, secretDataEncoder)
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

// Runs the editor; on error, we log a reason it failed. We always return the
// err (or nil if no error)
func runEditor(filename string) error {
	editor := whichEditor()

	cmd := exec.Command(editor, filename)
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(os.Stderr, "Editor %q exited: %s\n", editor, ee)
		} else {
			fmt.Fprintf(os.Stderr, "Error invoking %q: %s\n", editor, err)
		}
	}
	return err
}

func exitUsage(reason string) {
	if reason != "" {
		fmt.Fprintf(os.Stderr, "%s\n\n", reason)
	}
	fmt.Fprintf(os.Stderr, "Usage: %s filename.yml\n", os.Args[0])
	os.Exit(1)
}

// Gets the input filename. If not passed in, shows the usage and exits.
func getInputFilename() string {
	if len(os.Args) != 2 {
		exitUsage("Missing filename")
	}
	return os.Args[1]
}

func yamlFileTranslator(inFile *os.File, outFile *os.File, munger yamlSecretFileMunger) error {
	yamlInStr, err := ioutil.ReadAll(inFile)
	if err != nil {
		return err
	}

	yamlData := yaml.MapSlice{}
	err = yaml.Unmarshal(yamlInStr, &yamlData)
	if err != nil {
		return err
	}

	err = munger(yamlData)
	if err != nil {
		return err
	}

	yamlOutStr, err := yaml.Marshal(yamlData)
	if err != nil {
		return err
	}

	_, err = outFile.Write(yamlOutStr)
	return err
}

func main() {
	inFilename := getInputFilename()
	inFile, err := os.Open(inFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %s for reading: %s\n", inFilename, err)
		os.Exit(1)
	}
	defer func() {
		if inFile != nil {
			inFile.Close()
		}
	}()

	tmpFile, err := ioutil.TempFile("", "ksed")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening a temporary file for writing: %s\n", err)
		os.Exit(1)
	}
	tmpFilename := tmpFile.Name() // We'll need to reopen it later.
	defer func() {
		if tmpFile != nil {
			tmpFile.Close()
		}
		os.Remove(tmpFilename)
	}()

	err = yamlFileTranslator(inFile, tmpFile, decodeSecretData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding secret file: %s\n", err)
		os.Exit(1)
	}

	// Okay, before we run our editor, we'll close all FDs to ensure there's no issues
	// or cached data when we next read. We replace their values with nil so we crash if
	// we attempt to use them before they're reopened.
	inFile.Close()
	inFile = nil
	tmpFile.Close()
	tmpFile = nil

	if err = runEditor(tmpFilename); err != nil {
		os.Exit(1) // Error printed for us by runEditor
	}

	// Okay, edit was complete, let's reopen and convert back.
	if inFile, err = os.OpenFile(inFilename, os.O_RDWR, 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error re-opening %s for write: %s\n", inFilename, err)
		os.Exit(1)
	}
	if tmpFile, err = os.Open(tmpFilename); err != nil {
		fmt.Fprintf(os.Stderr, "Error re-opening %s for read: %s\n", tmpFilename, err)
		os.Exit(1)
	}

	// Okay, all's reopened, so let's encode those secrets again
	err = yamlFileTranslator(tmpFile, inFile, encodeSecretData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding secret file: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("All done!\n")
}
