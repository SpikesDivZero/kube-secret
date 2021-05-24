package pkg

import (
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type KubeSecretMunger interface {
	SetDebug(bool)

	// Use this first to read in data
	ReadFrom(io.Reader) error

	// Either encode or decode the b64 secrets, in-place.
	EncodeSecrets() error
	DecodeSecrets() error

	// Then write your data out somewhere
	WriteTo(io.Writer) error
}

type kubeSecretMunger struct {
	debug bool

	data       yaml.MapSlice
	dataLoaded bool
}

func NewKubeSecretMunger() KubeSecretMunger {
	return &kubeSecretMunger{}
}

func (k *kubeSecretMunger) SetDebug(d bool) {
	k.debug = d
}

// Reads in and unmarshals the YAML. If we hit an error, or the file is not a
// secret (as indicated by `kind: Secret`), then we return an error.
func (k *kubeSecretMunger) ReadFrom(r io.Reader) error {
	yamlInStr, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	k.data = yaml.MapSlice{}
	err = yaml.Unmarshal(yamlInStr, &k.data)
	if err != nil {
		return err
	}

	err = k.ensureIsSecret()
	if err != nil {
		return err
	}

	k.dataLoaded = true
	return nil
}

func (k *kubeSecretMunger) EncodeSecrets() error {
	if !k.dataLoaded {
		panic("Invalid State: Data not loaded")
	}

	return processSecretsInYaml(k.data, secretDataEncoder)
}

func (k *kubeSecretMunger) DecodeSecrets() error {
	if !k.dataLoaded {
		panic("Invalid State: Data not loaded")
	}

	return processSecretsInYaml(k.data, secretDataDecoder)
}

func (k *kubeSecretMunger) WriteTo(w io.Writer) error {
	if !k.dataLoaded {
		panic("Invalid State: Data not loaded")
	}

	yamlOutStr, err := yaml.Marshal(k.data)
	if err != nil {
		return err
	}

	_, err = w.Write(yamlOutStr)
	return err
}

//
//
//
//
// Old code below here. Should probably be updated.
//
//
//
//

// Returns a pointer to the MapItem with the corresponding Key.
func findKey(ms yaml.MapSlice, key string) *yaml.MapItem {
	for idx := range ms {
		kv := &ms[idx]
		if kv.Key == key {
			return kv
		}
	}
	return nil
}

func (k *kubeSecretMunger) ensureIsSecret() error {
	kv := findKey(k.data, "kind")
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
		return fmt.Errorf("Secret %q is %#v, expected string", kv.Key, kv.Value)
	}

	decoded, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return fmt.Errorf("Secret %q is %#v, failed to decode base64", kv.Key, kv.Value)
	}

	kv.Value = string(decoded)
	return nil
}

func secretDataEncoder(kv *yaml.MapItem) error {
	secret, ok := kv.Value.(string)
	if !ok {
		return fmt.Errorf("Secret %q is %#v, expected string", kv.Key, kv.Value)
	}

	fmt.Printf("Secret %q is %#v\n", kv.Key, kv.Value)

	kv.Value = base64.StdEncoding.EncodeToString([]byte(secret))
	return nil
}

// Applies a secretDataMunger to all values in .data
func processSecretsInYaml(data yaml.MapSlice, fn secretDataMunger) error {
	kv := findKey(data, "data")
	if kv == nil {
		return fmt.Errorf("Yaml file does not have a `data`?")
	}

	secretData, ok := kv.Value.(yaml.MapSlice)
	if !ok {
		return fmt.Errorf("Yaml file `data` is %#v, expected MapSlice", kv.Value)
	}

	for idx := range secretData {
		if err := fn(&secretData[idx]); err != nil {
			return err
		}
	}

	return nil
}
