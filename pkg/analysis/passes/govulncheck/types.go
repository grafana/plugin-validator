package govulncheck

// Message is one record from `govulncheck -json` stream output.
// Each line is a single Message with exactly one of the embedded fields populated.
type Message struct {
	Config   *Config   `json:"config,omitempty"`
	Progress *Progress `json:"progress,omitempty"`
	OSV      *OSV      `json:"osv,omitempty"`
	Finding  *Finding  `json:"finding,omitempty"`
}

type Config struct {
	ProtocolVersion string `json:"protocol_version,omitempty"`
	ScannerName     string `json:"scanner_name,omitempty"`
	ScannerVersion  string `json:"scanner_version,omitempty"`
	DB              string `json:"db,omitempty"`
	DBLastModified  string `json:"db_last_modified,omitempty"`
	GoVersion       string `json:"go_version,omitempty"`
	ScanLevel       string `json:"scan_level,omitempty"`
}

type Progress struct {
	Message string `json:"message,omitempty"`
}

type OSV struct {
	ID      string `json:"id,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// Finding reports one vulnerability hit. Trace describes the call stack from
// user code into the vulnerable symbol; for non-called findings (module- or
// package-level), the user-code frames are absent.
type Finding struct {
	OSV          string  `json:"osv,omitempty"`
	FixedVersion string  `json:"fixed_version,omitempty"`
	Trace        []Frame `json:"trace,omitempty"`
}

type Frame struct {
	Module   string `json:"module,omitempty"`
	Version  string `json:"version,omitempty"`
	Package  string `json:"package,omitempty"`
	Function string `json:"function,omitempty"`
	Receiver string `json:"receiver,omitempty"`
	Position *Pos   `json:"position,omitempty"`
}

type Pos struct {
	Filename string `json:"filename,omitempty"`
	Line     int    `json:"line,omitempty"`
}
