package metadataschema

import (
	"io"
	"net/http"

	"github.com/grafana/plugin-validator/pkg/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "metadataschema",
	Run:  run,
}

const schemaURL = "https://raw.githubusercontent.com/grafana/grafana/main/docs/sources/developers/plugins/plugin.schema.json"

func run(_ *analysis.Pass) (interface{}, error) {
	resp, err := http.Get(schemaURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, err
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return b, nil
}
