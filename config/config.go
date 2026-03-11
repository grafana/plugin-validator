package config

import (
	_ "embed"
)

//go:embed grafana.yaml
var GrafanaConfig []byte
