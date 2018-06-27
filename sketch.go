package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"

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

func main() {

	yamlInput, err := ioutil.ReadFile("fake-secret.yaml")
	if err != nil {
		panic(err)
	}

	fmt.Println("# Input yaml file:")
	fmt.Println(string(yamlInput))

	yamlData := yaml.MapSlice{}
	err = yaml.Unmarshal(yamlInput, &yamlData)
	if err != nil {
		panic(err)
	}

	err = decodeSecretData(yamlData)
	if err != nil {
		panic(err)
	}

	yamlOutput, err := yaml.Marshal(yamlData)
	if err != nil {
		panic(err)
	}

	fmt.Println("# Decoded yaml file:")
	fmt.Println(string(yamlOutput))

	encodeSecretData(yamlData)
	yamlOutput, err = yaml.Marshal(yamlData)
	if err != nil {
		panic(err)
	}

	fmt.Println("# Encoded yaml file:")
	fmt.Println(string(yamlOutput))
}
