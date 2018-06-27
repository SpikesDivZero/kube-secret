package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

func main() {
	b64 := base64.StdEncoding

	yamlInput, err := ioutil.ReadFile("fake-secret.yaml")
	if err != nil {
		panic(err)
	}

	yamlData := yaml.MapSlice{}
	err = yaml.Unmarshal(yamlInput, &yamlData)
	if err != nil {
		panic(err)
	}

	// Pointers so we can modify the structs in place, without keeping track of
	// arbitrary indexes.
	var miKind *yaml.MapItem
	var miData *yaml.MapItem

	for idx, kv := range yamlData {
		if kv.Key == "kind" {
			miKind = &yamlData[idx]
		} else if kv.Key == "data" {
			miData = &yamlData[idx]
		}
	}

	if miKind == nil {
		panic("Yaml file does not have a `kind`?")
	}
	kind, ok := miKind.Value.(string)
	if !ok {
		panic(fmt.Sprintf("Yaml file `kind` is %#v, expected string", miKind.Value))
	}
	if kind != "Secret" {
		panic(fmt.Sprintf("Yaml file `kind` is %q, expected 'Secret'", kind))
	}

	if miData == nil {
		panic("Yaml file does not have a `data`?")
	}
	data, ok := miData.Value.(yaml.MapSlice)
	if !ok {
		panic(fmt.Sprintf("Yaml file `data` is %#v, expected MapSlice", miData.Value))
	}
	for idx, kv := range data {
		secret, ok := kv.Value.(string)
		if !ok {
			panic(fmt.Sprintf("Secret %q is %#v, expected string", kv.Value))
		}

		decoded, err := b64.DecodeString(secret)
		if err != nil {
			panic(fmt.Sprintf("Secret $q is %#v, failed to decode base64", kv.Value))
		}

		// Must use idx to write, so we don't write to the copy
		data[idx].Value = string(decoded)
	}

	fmt.Println("# Decoded yaml file:")
	yamlOutput, err := yaml.Marshal(yamlData)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(yamlOutput))
}
