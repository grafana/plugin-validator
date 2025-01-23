package provenance

import "time"

type AttestationResponse struct {
	VerificationResult VerificationResult `json:"verificationResult"`
}

type VerificationResult struct {
	MediaType          string              `json:"mediaType"`
	Statement          Statement           `json:"statement"`
	VerifiedTimestamps []VerifiedTimestamp `json:"verifiedTimestamps"`
}

type VerifiedTimestamp struct {
	Type      string    `json:"type"`
	URI       string    `json:"uri"`
	Timestamp time.Time `json:"timestamp"`
}

type Statement struct {
	Type      string    `json:"type"`
	Subject   []Subject `json:"subject"`
	Predicate Predicate `json:"predicate"`
}

type Subject struct {
	Name   string            `json:"name"`
	Digest map[string]string `json:"digest"`
}

type Predicate struct {
	RunDetails RunDetails `json:"runDetails"`
}

type RunDetails struct {
	Builder  Builder  `json:"builder"`
	Metadata Metadata `json:"metadata"`
}

type Builder struct {
	ID string `json:"id"`
}

type Metadata struct {
	InvocationID string `json:"invocationId"`
}
