package metadata

type Metadata struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Type       string     `json:"type"`
	Info       Info       `json:"info"`
	Includes   []Includes `json:"includes"`
	Executable string     `json:"executable"`
	Backend    bool       `json:"backend"`
	Alerting   bool       `json:"alerting"`
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
	URL string `json:"url"`
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
