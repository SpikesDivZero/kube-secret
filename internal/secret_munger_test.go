package internal

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

var testDataFS = func() fs.FS {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return os.DirFS(filepath.Join(wd, "..", "test-data"))
}()

func TestNewKubeSecretMunger(t *testing.T) {
	ki := NewKubeSecretMunger()
	k, ok := ki.(*kubeSecretMunger)
	require.True(t, ok, "returns an instance of *kubeSecretMunger")
	assert.False(t, k.dataLoaded, "data is not yet loaded at creation")
}

func Test_kubeSecretMunger_ReadFrom(t *testing.T) {
	getReader := func(filename string) io.Reader {
		data, err := fs.ReadFile(testDataFS, filename)
		require.NoError(t, err, "our test filenames should always exist")

		return bytes.NewReader(data)
	}

	tests := []struct {
		name    string
		reader  io.Reader
		wantErr bool
	}{
		{
			"invalid-yaml fails parse",
			getReader("invalid-yaml.yaml"),
			true,
		},
		{
			"not-a-secret fails validation",
			getReader("not-a-secret.yaml"),
			true,
		},
		{
			"fake-secret works",
			getReader("fake-secret.yaml"),
			false,
		},
		{
			"read error",
			iotest.ErrReader(errors.New("something")),
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &kubeSecretMunger{}

			if err := k.ReadFrom(tt.reader); (err != nil) != tt.wantErr {
				t.Errorf("kubeSecretMunger.ReadFrom() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			assert.True(t, k.dataLoaded, "dataLoaded flag should turn true")
		})
	}
}

type bustedWriter struct {
	bytes.Buffer
}

func (bw *bustedWriter) Write(p []byte) (int, error) {
	if len(p) < 2 {
		return 0, nil
	}
	return 1, io.ErrShortWrite
}

func Test_kubeSecretMunger_WriteTo(t *testing.T) {
	// The actual data doesn't matter too much, so this works as a stub for most tests.
	simpleData := yaml.MapSlice{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: "value2"},
	}
	simpleOutputBytes, err := yaml.Marshal(simpleData)
	require.NoError(t, err)
	simpleOutput := string(simpleOutputBytes)

	tests := []struct {
		name    string
		writer  io.Writer
		data    yaml.MapSlice
		wantW   string
		wantErr bool
	}{
		{
			"data not loaded",
			&bytes.Buffer{},
			nil,
			"",
			true,
		},
		{
			"happy path",
			&bytes.Buffer{},
			simpleData,
			simpleOutput,
			false,
		},
		{
			"short write",
			&bustedWriter{},
			simpleData,
			"",
			true,
		},
		// TODO: I don't have any good test cases to trip yaml.Marshal into returning an error
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &kubeSecretMunger{
				data:       tt.data,
				dataLoaded: tt.data != nil,
			}
			if err := k.WriteTo(tt.writer); (err != nil) != tt.wantErr {
				t.Errorf("kubeSecretMunger.WriteTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}

			// Non-error cases should always use bytes.Buffer
			bb, ok := tt.writer.(*bytes.Buffer)
			require.True(t, ok, "non-error test case using something other than bytes.Buffer")
			if gotW := bb.String(); gotW != tt.wantW {
				t.Errorf("kubeSecretMunger.WriteTo() = %v, want %v", gotW, tt.wantW)
			}
		})
	}
}

func Test_findKey(t *testing.T) {
	ms := yaml.MapSlice{
		{Key: "key2", Value: "value2"},
		{Key: "keyX", Value: "boop"},
		{Key: "key1", Value: "value1"},
	}

	tests := []struct {
		key  string
		want *yaml.MapItem
	}{
		{"key1", &ms[2]},
		{"key2", &ms[0]},
		{"dne", nil},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := findKey(ms, tt.key); got != tt.want { // Pointer comparison
				t.Errorf("findKey() = %v (at %p), want %v (at %p)", got, got, tt.want, tt.want)
			}
		})
	}
}

func Test_kubeSecretMunger_ensureIsSecret(t *testing.T) {
	tests := []struct {
		name    string
		data    yaml.MapSlice
		wantErr bool
	}{
		{
			"not loaded",
			nil,
			true,
		},
		{
			"no kind",
			yaml.MapSlice{},
			true,
		},
		{
			"kind is not string",
			yaml.MapSlice{
				{Key: "kind", Value: 42},
			},
			true,
		},
		{
			"kind is not secret",
			yaml.MapSlice{
				{Key: "kind", Value: "junk"},
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &kubeSecretMunger{
				data:       tt.data,
				dataLoaded: tt.data != nil,
			}
			if err := k.ensureIsSecret(); (err != nil) != tt.wantErr {
				t.Errorf("kubeSecretMunger.ensureIsSecret() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_secretDataDecoder(t *testing.T) {
	tests := []struct {
		name    string
		kv      *yaml.MapItem
		wantErr bool
		wantKv  *yaml.MapItem
	}{
		{
			"happy path",
			&yaml.MapItem{Key: "keyA", Value: "Ym9vcA=="},
			false,
			&yaml.MapItem{Key: "keyA", Value: "boop"},
		},
		{
			"invalid base46",
			&yaml.MapItem{Key: "keyB", Value: "invalid!"},
			true,
			nil,
		},
		{
			"bad type",
			&yaml.MapItem{Key: "key2", Value: yaml.MapSlice{}},
			true,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := secretDataDecoder(tt.kv); (err != nil) != tt.wantErr {
				t.Errorf("secretDataDecoder() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}
			assert.Equal(t, tt.wantKv, tt.kv, "kv decoded correctly") // checks values, not pointers
		})
	}
}

func Test_secretDataEncoder(t *testing.T) {
	tests := []struct {
		name    string
		kv      *yaml.MapItem
		wantErr bool
		wantKv  *yaml.MapItem
	}{
		{
			"happy path",
			&yaml.MapItem{Key: "key1", Value: "goopy"},
			false,
			&yaml.MapItem{Key: "key1", Value: "Z29vcHk="},
		},
		{
			"bad type",
			&yaml.MapItem{Key: "key2", Value: yaml.MapSlice{}},
			true,
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := secretDataEncoder(tt.kv); (err != nil) != tt.wantErr {
				t.Errorf("secretDataEncoder() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}
			assert.Equal(t, tt.wantKv, tt.kv, "kv encoded correctly") // checks values, not pointers
		})
	}
}

// TODO: No tests currently for EncodeSecrets or DecodeSecrets, since they're slim wrappers over processSecretsInYaml

func Test_processSecretsInYaml(t *testing.T) {
	seenKeys := []string{}
	keyRecorder := func(kv *yaml.MapItem) error {
		seenKeys = append(seenKeys, kv.Key.(string))
		if kv.Key == "trigger-error" {
			return errors.New("error")
		}
		return nil
	}

	tests := []struct {
		name     string
		data     yaml.MapSlice
		wantErr  bool
		wantKeys []string
	}{
		{
			"no data", // could be stringData
			yaml.MapSlice{},
			false,
			[]string{},
		},
		{
			"data wrong type",
			yaml.MapSlice{
				yaml.MapItem{Key: "data", Value: 42},
			},
			true,
			nil,
		},
		{
			"callback errors",
			yaml.MapSlice{
				yaml.MapItem{Key: "data", Value: yaml.MapSlice{
					yaml.MapItem{Key: "trigger-error", Value: ""},
				}},
			},
			true,
			nil,
		},
		{
			"happy path",
			yaml.MapSlice{
				yaml.MapItem{Key: "data", Value: yaml.MapSlice{
					yaml.MapItem{Key: "secret1", Value: "hai"},
					yaml.MapItem{Key: "something", Value: "boop"},
					yaml.MapItem{Key: "secret2", Value: "bai"},
				}},
			},
			false,
			[]string{"secret1", "something", "secret2"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seenKeys = []string{}
			if err := processSecretsInYaml(tt.data, keyRecorder); (err != nil) != tt.wantErr {
				t.Errorf("processSecretsInYaml() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}
			assert.Equal(t, tt.wantKeys, tt.wantKeys)
		})
	}
}
