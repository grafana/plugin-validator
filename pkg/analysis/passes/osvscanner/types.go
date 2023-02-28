package osvscanner

import "time"

/*
Possible Severity levels returned by osv-scanner:

	CRITICAL
	HIGH
	MODERATE
	LOW
*/

const (
	SeverityCritical = "CRITICAL"
	SeverityHigh     = "HIGH"
	SeverityModerate = "MODERATE"
	SeverityLow      = "LOW"
)

// OSVJsonOutput output expected by osv-scanner as a struct
type OSVJsonOutput struct {
	Results []struct {
		Source struct {
			Path string `json:"path"`
			Type string `json:"type"`
		} `json:"source"`
		Packages []struct {
			Package struct {
				Name      string `json:"name"`
				Version   string `json:"version"`
				Ecosystem string `json:"ecosystem"`
			} `json:"package"`
			Vulnerabilities []struct {
				SchemaVersion string    `json:"schema_version"`
				ID            string    `json:"id"`
				Modified      time.Time `json:"modified"`
				Published     time.Time `json:"published"`
				Aliases       []string  `json:"aliases"`
				Summary       string    `json:"summary"`
				Details       string    `json:"details"`
				Affected      []struct {
					Package struct {
						Ecosystem string `json:"ecosystem"`
						Name      string `json:"name"`
						Purl      string `json:"purl"`
					} `json:"package"`
					Ranges []struct {
						Type   string `json:"type"`
						Events []struct {
							Introduced string `json:"introduced,omitempty"`
							Fixed      string `json:"fixed,omitempty"`
						} `json:"events"`
					} `json:"ranges"`
					DatabaseSpecific struct {
						Source string `json:"source"`
						URL    string `json:"url,omitempty"`
					} `json:"database_specific"`
				} `json:"affected"`
				References []struct {
					Type string `json:"type"`
					URL  string `json:"url"`
				} `json:"references"`
				DatabaseSpecific struct {
					CweIds         []string `json:"cwe_ids,omitempty"`
					GithubReviewed bool     `json:"github_reviewed,omitempty"`
					Severity       string   `json:"severity,omitempty"`
				} `json:"database_specific"`
			} `json:"vulnerabilities"`
			Groups []struct {
				Ids []string `json:"ids"`
			} `json:"groups"`
		} `json:"packages"`
	} `json:"results"`
}

var GrafanaToolkitPackages = map[string]bool{
	"axios":                           true,
	"babel-jest":                      true,
	"babel-loader":                    true,
	"babel-plugin-angularjs-annotate": true,
	"chalk":                           true,
	"command-exists":                  true,
	"commander":                       true,
	"copy-webpack-plugin":             true,
	"css-loader":                      true,
	"css-minimizer-webpack-plugin":    true,
	"eslint":                          true,
	"eslint-config-prettier":          true,
	"eslint-plugin-jsdoc":             true,
	"eslint-plugin-react":             true,
	"eslint-plugin-react-hooks":       true,
	"execa":                           true,
	"file-loader":                     true,
	"fork-ts-checker-webpack-plugin":  true,
	"fs-extra":                        true,
	"globby":                          true,
	"html-loader":                     true,
	"html-webpack-plugin":             true,
	"inquirer":                        true,
	"jest":                            true,
	"jest-canvas-mock":                true,
	"jest-junit":                      true,
	"less":                            true,
	"less-loader":                     true,
	"lodash":                          true,
	"md5-file":                        true,
	"mini-css-extract-plugin":         true,
	"ora":                             true,
	"pixelmatch":                      true,
	"pngjs":                           true,
	"postcss":                         true,
	"postcss-flexbugs-fixes":          true,
	"postcss-loader":                  true,
	"postcss-preset-env":              true,
	"prettier":                        true,
	"react-dev-utils":                 true,
	"replace-in-file-webpack-plugin":  true,
	"rimraf":                          true,
	"sass":                            true,
	"sass-loader":                     true,
	"semver":                          true,
	"simple-git":                      true,
	"style-loader":                    true,
	"terser-webpack-plugin":           true,
	"ts-jest":                         true,
	"ts-loader":                       true,
	"ts-node":                         true,
	"tslib":                           true,
	"typescript":                      true,
	"url-loader":                      true,
	"webpack":                         true,
}

var GrafanaDataPackages = map[string]bool{
	"@braintree/sanitize-url": true,
	"@grafana/schema":         true,
	"@types/d3-interpolate":   true,
	"d3-interpolate":          true,
	"date-fns":                true,
	"eventemitter3":           true,
	"lodash":                  true,
	"marked":                  true,
	"moment":                  true,
	"moment-timezone":         true,
	"ol":                      true,
	"papaparse":               true,
	"react":                   true,
	"react-dom":               true,
	"regenerator-runtime":     true,
	"rxjs":                    true,
	"tslib":                   true,
	"uplot":                   true,
	"xss":                     true,
}

var GrafanaUIPackages = map[string]bool{
	"@emotion/css":              true,
	"@emotion/react":            true,
	"@grafana/data":             true,
	"@grafana/e2e-selectors":    true,
	"@grafana/schema":           true,
	"@grafana/slate-react":      true,
	"@monaco-editor/react":      true,
	"@popperjs/core":            true,
	"@react-aria/button":        true,
	"@react-aria/dialog":        true,
	"@react-aria/focus":         true,
	"@react-aria/menu":          true,
	"@react-aria/overlays":      true,
	"@react-aria/utils":         true,
	"@react-stately/menu":       true,
	"@sentry/browser":           true,
	"ansicolor":                 true,
	"calculate-size":            true,
	"classnames":                true,
	"core-js":                   true,
	"d3":                        true,
	"date-fns":                  true,
	"hoist-non-react-statics":   true,
	"immutable":                 true,
	"is-hotkey":                 true,
	"jquery":                    true,
	"lodash":                    true,
	"memoize-one":               true,
	"moment":                    true,
	"monaco-editor":             true,
	"ol":                        true,
	"prismjs":                   true,
	"rc-cascader":               true,
	"rc-drawer":                 true,
	"rc-slider":                 true,
	"rc-time-picker":            true,
	"react":                     true,
	"react-beautiful-dnd":       true,
	"react-calendar":            true,
	"react-colorful":            true,
	"react-custom-scrollbars-2": true,
	"react-dom":                 true,
	"react-dropzone":            true,
	"react-highlight-words":     true,
	"react-hook-form":           true,
	"react-inlinesvg":           true,
	"react-popper":              true,
	"react-popper-tooltip":      true,
	"react-router-dom":          true,
	"react-select":              true,
	"react-select-event":        true,
	"react-table":               true,
	"react-transition-group":    true,
	"react-use":                 true,
	"react-window":              true,
	"rxjs":                      true,
	"slate":                     true,
	"slate-plain-serializer":    true,
	"tinycolor2":                true,
	"tslib":                     true,
	"uplot":                     true,
	"uuid":                      true,
}

var GrafanaE2EPackages = map[string]bool{
	"@babel/core":                   true,
	"@babel/preset-env":             true,
	"@cypress/webpack-preprocessor": true,
	"@grafana/e2e-selectors":        true,
	"@grafana/tsconfig":             true,
	"@mochajs/json-file-reporter":   true,
	"babel-loader":                  true,
	"blink-diff":                    true,
	"chrome-remote-interface":       true,
	"commander":                     true,
	"cypress":                       true,
	"cypress-file-upload":           true,
	"devtools-protocol":             true,
	"execa":                         true,
	"lodash":                        true,
	"mocha":                         true,
	"resolve-as-bin":                true,
	"rimraf":                        true,
	"tracelib":                      true,
	"ts-loader":                     true,
	"tslib":                         true,
	"typescript":                    true,
	"uuid":                          true,
	"yaml":                          true,
}

// CommonPackages Packages that are frequently flagged but will be ignored
var CommonPackages = map[string]bool{
	"underscore": true,
}
