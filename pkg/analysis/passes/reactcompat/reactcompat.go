package reactcompat

import (
	"bytes"
	"fmt"
	"regexp"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/modulejs"
)

const react19UpgradeGuide = "https://react.dev/blog/2024/04/25/react-19-upgrade-guide"

var (
	react19PropTypes       = &analysis.Rule{Name: "react-19-prop-types", Severity: analysis.Warning}
	react19LegacyContext   = &analysis.Rule{Name: "react-19-legacy-context", Severity: analysis.Warning}
	react19StringRefs      = &analysis.Rule{Name: "react-19-string-refs", Severity: analysis.Warning}
	react19CreateFactory   = &analysis.Rule{Name: "react-19-create-factory", Severity: analysis.Warning}
	react19FindDOMNode     = &analysis.Rule{Name: "react-19-find-dom-node", Severity: analysis.Warning}
	react19LegacyRender    = &analysis.Rule{Name: "react-19-legacy-render", Severity: analysis.Warning}
	react19SecretInternals = &analysis.Rule{Name: "react-19-secret-internals", Severity: analysis.Warning}
	react19Compatible = &analysis.Rule{Name: "react-19-compatible", Severity: analysis.OK}
)

var Analyzer = &analysis.Analyzer{
	Name:     "reactcompat",
	Requires: []*analysis.Analyzer{modulejs.Analyzer},
	Run:      run,
	Rules: []*analysis.Rule{
		react19PropTypes,
		react19LegacyContext,
		react19StringRefs,
		react19CreateFactory,
		react19FindDOMNode,
		react19LegacyRender,
		react19SecretInternals,
		react19Compatible,
	},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "React 19 Compatibility",
		Description: "Detects usage of React APIs removed or deprecated in React 19.",
	},
}

// detector checks a single module.js file for a specific pattern.
type detector interface {
	Detect(moduleJs []byte) bool
	Pattern() string
}

type containsBytesDetector struct {
	pattern []byte
}

func (d *containsBytesDetector) Detect(moduleJs []byte) bool {
	return bytes.Contains(moduleJs, d.pattern)
}

func (d *containsBytesDetector) Pattern() string {
	return string(d.pattern)
}

type regexDetector struct {
	regex *regexp.Regexp
}

func (d *regexDetector) Detect(moduleJs []byte) bool {
	return d.regex.Match(moduleJs)
}

func (d *regexDetector) Pattern() string {
	return d.regex.String()
}

// reactPattern groups a rule, a human-readable description, and the detectors that trigger it.
type reactPattern struct {
	rule        *analysis.Rule
	title       string
	description string
	detectors   []detector
}

var reactPatterns = []reactPattern{
	{
		rule:        react19PropTypes,
		title:       "module.js: Uses removed React API propTypes or defaultProps",
		description: "Detected usage of '%s'. propTypes and defaultProps on function components were removed in React 19.",
		detectors: []detector{
			&containsBytesDetector{pattern: []byte(".propTypes=")},
			&containsBytesDetector{pattern: []byte(".defaultProps=")},
		},
	},
	{
		rule:        react19LegacyContext,
		title:       "module.js: Uses removed React legacy context API",
		description: "Detected usage of '%s'. contextTypes, childContextTypes, and getChildContext were removed in React 19.",
		detectors: []detector{
			&containsBytesDetector{pattern: []byte(".contextTypes=")},
			&containsBytesDetector{pattern: []byte(".childContextTypes=")},
			&containsBytesDetector{pattern: []byte("getChildContext")},
		},
	},
	{
		rule:        react19StringRefs,
		title:       "module.js: Uses removed React string refs",
		description: "Detected usage of '%s'. String refs were removed in React 19. Use callback refs or React.createRef() instead.",
		detectors: []detector{
			&regexDetector{regex: regexp.MustCompile(`ref:"[^"]+?"`)},
			&regexDetector{regex: regexp.MustCompile(`ref:'[^']+'`)},
		},
	},
	{
		rule:        react19CreateFactory,
		title:       "module.js: Uses removed React.createFactory",
		description: "Detected usage of '%s'. React.createFactory was removed in React 19. Use JSX instead.",
		detectors: []detector{
			&containsBytesDetector{pattern: []byte("createFactory(")},
		},
	},
	{
		rule:        react19FindDOMNode,
		title:       "module.js: Uses removed ReactDOM.findDOMNode",
		description: "Detected usage of '%s'. ReactDOM.findDOMNode was removed in React 19. Use DOM refs instead.",
		detectors: []detector{
			&containsBytesDetector{pattern: []byte("findDOMNode(")},
		},
	},
	{
		rule:        react19LegacyRender,
		title:       "module.js: Uses removed ReactDOM.render or unmountComponentAtNode",
		description: "Detected usage of '%s'. ReactDOM.render and unmountComponentAtNode were removed in React 19. Use createRoot instead.",
		detectors: []detector{
			&containsBytesDetector{pattern: []byte("ReactDOM.render(")},
			&containsBytesDetector{pattern: []byte("unmountComponentAtNode(")},
		},
	},
	{
		rule:        react19SecretInternals,
		title:       "module.js: Uses React internal __SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED",
		description: "Detected usage of '%s'. This internal was removed in React 19.",
		detectors: []detector{
			&containsBytesDetector{pattern: []byte("__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED")},
		},
	},
}

func run(pass *analysis.Pass) (interface{}, error) {
	moduleJsMap, ok := pass.ResultOf[modulejs.Analyzer].(map[string][]byte)
	if !ok || len(moduleJsMap) == 0 {
		return nil, nil
	}

	for _, pattern := range reactPatterns {
		matched := false
		matchedPattern := ""

	outer:
		for _, content := range moduleJsMap {
			for _, d := range pattern.detectors {
				if d.Detect(content) {
					matched = true
					matchedPattern = d.Pattern()
					break outer
				}
			}
		}

		if matched {
			pass.ReportResult(
				pass.AnalyzerName,
				pattern.rule,
				pattern.title,
				fmt.Sprintf(pattern.description+" See: "+react19UpgradeGuide, matchedPattern),
			)
		}
	}

	return nil, nil
}
