package signature

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

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/manifest"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/clearsign"
)

var Analyzer = &analysis.Analyzer{
	Name:     "signature",
	Requires: []*analysis.Analyzer{manifest.Analyzer, metadata.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir := pass.ResultOf[archive.Analyzer].(string)
	metadata := pass.ResultOf[metadata.Analyzer].([]byte)
	manifest := pass.ResultOf[manifest.Analyzer].([]byte)

	var data struct {
		ID   string `json:"id"`
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.Unmarshal(metadata, &data); err != nil {
		return nil, nil
	}

	state := getPluginSignatureState(data.ID, data.Info.Version, archiveDir, manifest)

	switch state {
	case PluginSignatureUnsigned:
		pass.Report(analysis.Diagnostic{
			Severity: analysis.Error,
			Message:  "unsigned plugin",
		})
	case PluginSignatureInvalid:
		pass.Report(analysis.Diagnostic{
			Severity: analysis.Error,
			Message:  "invalid plugin signature",
			Context:  "MANIFEST.txt",
		})
	case PluginSignatureModified:
		pass.Report(analysis.Diagnostic{
			Severity: analysis.Error,
			Message:  "plugin has been modified since it was signed",
			Context:  "MANIFEST.txt",
		})
	default:
	}

	if state != PluginSignatureUnsigned {
		m, err := readPluginManifest(manifest)
		if err != nil {
			return nil, err
		}

		if m.SignatureType == "private" {
			pass.Report(analysis.Diagnostic{
				Severity: analysis.Error,
				Message:  "should be signed under community or commercial signature level",
				Context:  "MANIFEST.txt",
			})
		}
	}

	return nil, nil
}

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

func getPluginSignatureState(pluginID, version, pluginDir string, byteValue []byte) PluginSignature {
	if len(byteValue) < 10 {
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
	Version       string            `json:"version"`
	KeyID         string            `json:"keyId"`
	Time          int64             `json:"time"`
	Files         map[string]string `json:"files"`
	SignatureType string            `json:"signatureType"`
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
