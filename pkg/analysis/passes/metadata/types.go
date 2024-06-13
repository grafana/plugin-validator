package metadata

type Metadata struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Type       string             `json:"type"`
	Info       MetadataInfo       `json:"info"`
	Includes   []MetatadaIncludes `json:"includes"`
	Executable string             `json:"executable"`
	Backend    bool               `json:"backend"`
	Alerting   bool               `json:"alerting"`
}

type MetadataInfo struct {
	Author      MetadataAuthor        `json:"author"`
	Screenshots []MetadataScreenshots `json:"screenshots"`
	Logos       MetadataLogos         `json:"logos"`
	Links       []MetadataLink        `json:"links"`
	Version     string                `json:"version"`
	Keywords    []string              `json:"keywords"`
	Description string                `json:"description"`
}

type MetadataAuthor struct {
	URL string `json:"url"`
}

type MetadataScreenshots struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type MetadataLogos struct {
	Small string `json:"small"`
	Large string `json:"large"`
}

type MetadataLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type MetatadaIncludes struct {
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
