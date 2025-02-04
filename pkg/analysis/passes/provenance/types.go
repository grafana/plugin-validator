package provenance

type AttestationCollection struct {
	Attestations []Attestation `json:"attestations"`
}

type Attestation struct {
	Bundle       Bundle `json:"bundle"`
	RepositoryID int64  `json:"repository_id"`
	BundleURL    string `json:"bundle_url"`
}

type Bundle struct {
	MediaType            string               `json:"mediaType"`
	DSSTEnvelope         DSSTEnvelope         `json:"dsseEnvelope"`
	VerificationMaterial VerificationMaterial `json:"verificationMaterial"`
}

type DSSTEnvelope struct {
	Payload     string      `json:"payload"`
	PayloadType string      `json:"payloadType"`
	Signatures  []Signature `json:"signatures"`
}

type Signature struct {
	Sig string `json:"sig"`
}

type VerificationMaterial struct {
	TlogEntries               []TlogEntry            `json:"tlogEntries"`
	TimestampVerificationData map[string]interface{} `json:"timestampVerificationData"`
	Certificate               Certificate            `json:"certificate"`
}

type TlogEntry struct {
	LogIndex          string           `json:"logIndex"`
	LogID             LogID            `json:"logId"`
	KindVersion       KindVersion      `json:"kindVersion"`
	IntegratedTime    string           `json:"integratedTime"`
	InclusionPromise  InclusionPromise `json:"inclusionPromise"`
	InclusionProof    InclusionProof   `json:"inclusionProof"`
	CanonicalizedBody string           `json:"canonicalizedBody"`
}

type LogID struct {
	KeyID string `json:"keyId"`
}

type KindVersion struct {
	Kind    string `json:"kind"`
	Version string `json:"version"`
}

type InclusionPromise struct {
	SignedEntryTimestamp string `json:"signedEntryTimestamp"`
}

type InclusionProof struct {
	LogIndex   string     `json:"logIndex"`
	RootHash   string     `json:"rootHash"`
	TreeSize   string     `json:"treeSize"`
	Hashes     []string   `json:"hashes"`
	Checkpoint Checkpoint `json:"checkpoint"`
}

type Checkpoint struct {
	Envelope string `json:"envelope"`
}

type Certificate struct {
	RawBytes string `json:"rawBytes"`
}
