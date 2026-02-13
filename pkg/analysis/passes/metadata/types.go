package metadata

import "strings"

type Metadata struct {
	ID           string               `json:"id"`
	Name         string               `json:"name"`
	Type         string               `json:"type"`
	Info         Info                 `json:"info"`
	Includes     []Includes           `json:"includes"`
	Executable   string               `json:"executable"`
	Backend      bool                 `json:"backend"`
	Alerting     bool                 `json:"alerting"`
	Dependencies MetadataDependencies `json:"dependencies"`
}

// IsGrafanaLabs returns true if the plugin was developed by Grafana Labs.
// A plugin is considered developed by Grafana Labs if either
// the author name is "Grafana Labs" or the org name in the slug is "grafana"
func (m Metadata) IsGrafanaLabs() bool {
	return strings.EqualFold(m.Info.Author.Name, "grafana labs") || strings.EqualFold(orgFromPluginID(m.ID), "grafana")
}

type Info struct {
	Author      Author        `json:"author"`
	Screenshots []Screenshots `json:"screenshots"`
	Logos       Logos         `json:"logos"`
	Links       []Link        `json:"links"`
	Version     string        `json:"version"`
	Keywords    []string      `json:"keywords"`
	Description string        `json:"description"`
}

type Author struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Screenshots struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Logos struct {
	Small string `json:"small"`
	Large string `json:"large"`
}

type Link struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Includes struct {
	Action     string `json:"action"`
	AddToNav   bool   `json:"addToNav"`
	Component  string `json:"component"`
	DefaultNav bool   `json:"defaultNav"`
	Icon       string `json:"icon"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	Role       string `json:"role"`
	Type       string `json:"type"`
	Uid        string `json:"uid"`
}

type MetadataDependencies struct {
	GrafanaDependency string                     `json:"grafanaDependency"`
	Plugins           []MetadataPluginDependency `json:"plugins"`
}

type MetadataPluginDependency struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// orgFromPluginID extracts and returns the organization prefix from a plugin ID by splitting on the first hyphen.
// Returns an empty string if the plugin ID is empty or invalid.
func orgFromPluginID(id string) string {
	parts := strings.SplitN(id, "-", 3)
	if len(parts) < 1 {
		return ""
	}
	return parts[0]
}
