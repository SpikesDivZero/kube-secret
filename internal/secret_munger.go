package internal

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

var ErrDataNotLoaded = errors.New("invalid state: data not loaded")

// KubeSecretMunger acts as a wrapper to encode or decode the base64-encoded data inside of a YAML-encoded secret.
//
// Generally, the usage is something like:
//
//     k := NewKubeSecretMunger()
//     k.ReadFrom(reader)
//     k.DecodeSecrets()  // Or encode
//     k.WriteTo(writer)
type KubeSecretMunger interface {
	// Use this first to read in data.
	ReadFrom(io.Reader) error

	// Either encode or decode the b64 secrets, in-place.
	EncodeSecrets() error
	DecodeSecrets() error

	// Then write your data out somewhere.
	WriteTo(io.Writer) error
}

type kubeSecretMunger struct {
	// We use MapSlice here as we want to preserve the order of all keys in a loaded kube yaml file. While kubectl might
	// not care about the order of lines changing, git does and I'd prefer not to have non-op edits in the history.
	data       yaml.MapSlice
	dataLoaded bool
}

func NewKubeSecretMunger() KubeSecretMunger {
	return &kubeSecretMunger{}
}

// Reads in and unmarshals the YAML. If we hit an error, or the file is not a secret (as indicated by `kind: Secret`),
// then we return an error.
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
		return ErrDataNotLoaded
	}

	return processSecretsInYaml(k.data, secretDataEncoder)
}

func (k *kubeSecretMunger) DecodeSecrets() error {
	if !k.dataLoaded {
		return ErrDataNotLoaded
	}

	return processSecretsInYaml(k.data, secretDataDecoder)
}

func (k *kubeSecretMunger) WriteTo(w io.Writer) error {
	if !k.dataLoaded {
		return ErrDataNotLoaded
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
		return errors.New("yaml does not have a `kind`")
	}

	kind, ok := kv.Value.(string)
	if !ok {
		return fmt.Errorf("yaml `kind` is %#v, expected string", kv.Value)
	}
	if kind != "Secret" {
		return fmt.Errorf("yaml `kind` is %q, expected 'Secret'", kind)
	}
	return nil
}

// A callback which munges a yaml.MapItem in-place.
type secretDataMunger func(kv *yaml.MapItem) error

func secretDataDecoder(kv *yaml.MapItem) error {
	secret, ok := kv.Value.(string)
	if !ok {
		return fmt.Errorf("secret %q is %#v, expected string", kv.Key, kv.Value)
	}

	decoded, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return fmt.Errorf("secret %q is %#v, failed to decode base64: %w", kv.Key, kv.Value, err)
	}

	kv.Value = string(decoded)
	return nil
}

func secretDataEncoder(kv *yaml.MapItem) error {
	secret, ok := kv.Value.(string)
	if !ok {
		return fmt.Errorf("secret %q is %#v, expected string", kv.Key, kv.Value)
	}

	kv.Value = base64.StdEncoding.EncodeToString([]byte(secret))
	return nil
}

// Applies a secretDataMunger to all values in .data
func processSecretsInYaml(data yaml.MapSlice, fn secretDataMunger) error {
	kv := findKey(data, "data")
	if kv == nil {
		// If there's no data key, then there's nothing for us to deobfuscate. (Could be stringData)
		return nil
	}

	secretData, ok := kv.Value.(yaml.MapSlice)
	if !ok {
		return fmt.Errorf("yaml file `data` is %#v, expected MapSlice", kv.Value)
	}

	for idx := range secretData {
		if err := fn(&secretData[idx]); err != nil {
			return err
		}
	}

	return nil
}
