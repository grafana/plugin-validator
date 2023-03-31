/**
This source code comes from here (it is not exported)
https://github.com/google/osv-scanner/pkg/lockfile/types.go
*/

package lockfile

type Dependency struct {
	Name    string
	Version string
}

type PackageDetails struct {
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Commit       string       `json:"commit,omitempty"`
	Ecosystem    Ecosystem    `json:"ecosystem,omitempty"`
	CompareAs    Ecosystem    `json:"compareAs,omitempty"`
	Dependencies []Dependency `json:"dependencies,omitempty"`
}

type Ecosystem string

type PackageDetailsParser = func(pathToLockfile string) ([]PackageDetails, error)
