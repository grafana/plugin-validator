package manifest

type File struct {
	Files         map[string]string `json:"files"`
	RootUrls      []string          `json:"rootUrls"`
	SignatureType string            `json:"signatureType"`
}
