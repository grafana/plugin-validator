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

	"golang.org/x/crypto/openpgp"           //nolint:staticcheck
	"golang.org/x/crypto/openpgp/clearsign" //nolint:staticcheck

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/manifest"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

var (
	unsignedPlugin    = &analysis.Rule{Name: "unsigned-plugin", Severity: analysis.Warning}
	modifiedSignature = &analysis.Rule{Name: "modified-signature", Severity: analysis.Warning}
	invalidSignature  = &analysis.Rule{Name: "invalid-signature", Severity: analysis.Warning}
	privateSignature  = &analysis.Rule{Name: "private-signature", Severity: analysis.Warning}
	validSignature    = &analysis.Rule{Name: "valid-signature", Severity: analysis.OK}
)

var Analyzer = &analysis.Analyzer{
	Name:     "signature",
	Requires: []*analysis.Analyzer{manifest.Analyzer, metadata.Analyzer},
	Run:      run,
	Rules: []*analysis.Rule{
		unsignedPlugin,
		modifiedSignature,
		invalidSignature,
		privateSignature,
		validSignature,
	},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Signature",
		Description: "Ensures the plugin has a valid signature.",
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok {
		return nil, nil
	}

	md, ok := pass.ResultOf[metadata.Analyzer].([]byte)
	if !ok {
		return nil, nil
	}

	mf, ok := pass.ResultOf[manifest.Analyzer].([]byte)
	if !ok {
		return nil, nil
	}

	var data struct {
		ID   string `json:"id"`
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.Unmarshal(md, &data); err != nil {
		return nil, nil
	}

	state, err := getPluginSignatureState(data.ID, data.Info.Version, archiveDir, mf)
	if err != nil {
		pass.ReportResult(
			pass.AnalyzerName,
			invalidSignature,
			"MANIFEST.txt: failed to check plugin signature",
			err.Error(),
		)
		return nil, nil
	}

	switch state {
	case PluginSignatureUnsigned:
		pass.ReportResult(pass.AnalyzerName, unsignedPlugin, "unsigned plugin", "")
	case PluginSignatureInvalid:
		pass.ReportResult(
			pass.AnalyzerName,
			invalidSignature,
			"MANIFEST.txt: invalid plugin signature",
			"The plugin might had been modified after it was signed.",
		)
	case PluginSignatureModified:
		pass.ReportResult(
			pass.AnalyzerName,
			modifiedSignature,
			"MANIFEST.txt: plugin has been modified since it was signed",
			"The plugin might had been modified after it was signed.",
		)
	default:
		if validSignature.ReportAll {
			pass.ReportResult(
				pass.AnalyzerName,
				validSignature,
				"MANIFEST.txt: valid plugin signature",
				"",
			)
		}
	}

	if state != PluginSignatureUnsigned {
		m, err := readPluginManifest(mf)
		if err != nil {
			return nil, err
		}

		if m.SignatureType == "private" {
			pass.ReportResult(
				pass.AnalyzerName,
				privateSignature,
				"MANIFEST.txt: plugin must be signed under community or commercial signature level",
				"The plugin is signed under private signature level.",
			)
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

var errFileHashMismatch = errors.New("file hash mismatch")

func checkFileSignature(fp string, expHash string) error {
	f, err := os.Open(fp)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return err
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if sum != expHash {
		return fmt.Errorf("%w: expected %q, got %q", errFileHashMismatch, expHash, sum)
	}
	return nil
}

func getPluginSignatureState(
	pluginID, version, pluginDir string,
	byteValue []byte,
) (PluginSignature, error) {
	if len(byteValue) < 10 {
		return PluginSignatureUnsigned, nil
	}

	m, err := readPluginManifest(byteValue)
	if err != nil {
		return PluginSignatureInvalid, nil
	}

	// Make sure the versions all match
	if m.Plugin != pluginID || m.Version != version {
		return PluginSignatureModified, nil
	}

	// Verify the manifest contents
	for p, hash := range m.Files {
		err = checkFileSignature(filepath.Join(pluginDir, p), hash)
		if errors.Is(err, errFileHashMismatch) {
			return PluginSignatureModified, nil
		}
		if err != nil {
			return 0, err
		}
	}
	// Everything OK
	return PluginSignatureValid, nil
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
	m := &pluginManifest{}
	err := json.Unmarshal(block.Plaintext, &m)
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

	return m, nil
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
