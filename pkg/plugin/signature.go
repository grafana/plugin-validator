package plugin

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/crypto/openpgp"           //lint:ignore SA1019 Ignore the deprecation warnings
	"golang.org/x/crypto/openpgp/clearsign" //lint:ignore SA1019 Ignore the deprecation warnings
)

type PluginSignature int

const (
	PluginSignatureUnsigned PluginSignature = iota
	PluginSignatureInvalid
	PluginSignatureModified
	PluginSignatureValid
)

type PluginBase struct {
	ID        string
	PluginDir string
	Version   string
}

func getPluginSignatureState(pluginID, version, pluginDir string) PluginSignature {
	manifestPath := filepath.Join(pluginDir, "MANIFEST.txt")

	byteValue, err := os.ReadFile(manifestPath)
	if err != nil || len(byteValue) < 10 {
		return PluginSignatureUnsigned
	}

	manifest, err := readPluginManifest(byteValue)
	if err != nil {
		return PluginSignatureInvalid
	}

	// Make sure the versions all match
	if manifest.Plugin != pluginID || manifest.Version != version {
		return PluginSignatureModified
	}

	// Verify the manifest contents
	for p, hash := range manifest.Files {
		// Open the file
		fp := filepath.Join(pluginDir, p)
		f, err := os.Open(fp)
		if err != nil {
			return PluginSignatureModified
		}
		defer f.Close()

		h := sha256.New()
		if _, err := io.Copy(h, f); err != nil {
			return PluginSignatureModified
		}
		sum := hex.EncodeToString(h.Sum(nil))
		if sum != hash {
			return PluginSignatureModified
		}
	}

	// Everything OK
	return PluginSignatureValid
}

// pluginManifest holds details for the file manifest
type pluginManifest struct {
	Plugin        string            `json:"plugin"`
	SignatureType string            `json:"signatureType"`
	Version       string            `json:"version"`
	KeyID         string            `json:"keyId"`
	Time          int64             `json:"time"`
	Files         map[string]string `json:"files"`
}

// readPluginManifest attempts to read and verify the plugin manifest
// if any error occurs or the manifest is not valid, this will return an error
func readPluginManifest(body []byte) (*pluginManifest, error) {
	block, _ := clearsign.Decode(body)
	if block == nil {
		return nil, errors.New("unable to decode manifest")
	}

	// Convert to a well typed object
	manifest := &pluginManifest{}
	err := json.Unmarshal(block.Plaintext, &manifest)
	if err != nil {
		return nil, fmt.Errorf("error parsing manifest JSON: %w", err)
	}

	publicKeyText, err := publicKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	keyring, err := openpgp.ReadArmoredKeyRing(bytes.NewBufferString(publicKeyText))
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	if _, err := openpgp.CheckDetachedSignature(keyring,
		bytes.NewBuffer(block.Bytes),
		block.ArmoredSignature.Body); err != nil {
		return nil, fmt.Errorf("failed to check signature: %w", err)
	}

	return manifest, nil
}

func publicKey() (string, error) {
	var data struct {
		Items []struct {
			KeyID  string `json:"keyId"`
			Since  int64  `json:"since"`
			Public string `json:"public"`
		}
	}

	resp, err := http.Get("https://grafana.com/api/plugins/ci/keys")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	if len(data.Items) == 0 {
		return "", errors.New("missing public key")
	}

	return data.Items[0].Public, nil
}
