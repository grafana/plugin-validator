package utils

import (
	"encoding/json"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/metadata"
)

func JSONToMetadata(jsonStr []byte) (metadata.Metadata, error) {
	// Create a variable to store the unmarshaled Metadata.
	var content metadata.Metadata

	// Unmarshal the JSON string into the Metadata struct.
	err := json.Unmarshal(jsonStr, &content)
	if err != nil {
		return metadata.Metadata{}, err
	}

	return content, nil
}

// RunDependencies runs the dependencies of an analyzer recursively.
// All the results are stored in the provided *analysis.Pass.
// If the result of an analyzer is already in the pass, it will not be run again.
func RunDependencies(pass *analysis.Pass, analyzer *analysis.Analyzer) error {
	for _, dep := range analyzer.Requires {
		if _, ok := pass.ResultOf[dep]; ok {
			continue
		}
		if err := RunDependencies(pass, dep); err != nil {
			return err
		}
		var err error
		pass.ResultOf[dep], err = dep.Run(pass)
		if err != nil {
			return err
		}
	}
	return nil
}
