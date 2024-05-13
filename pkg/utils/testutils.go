package utils

import (
	"encoding/json"

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
