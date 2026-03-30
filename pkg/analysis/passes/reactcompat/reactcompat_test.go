package reactcompat

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/modulejs"
	"github.com/grafana/plugin-validator/pkg/testpassinterceptor"
)

func newPass(interceptor *testpassinterceptor.TestPassInterceptor, content map[string][]byte) *analysis.Pass {
	return &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			modulejs.Analyzer: content,
		},
		Report: interceptor.ReportInterceptor(),
	}
}

func TestCleanPlugin(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`import { PanelPlugin } from '@grafana/data'`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	// No warnings; the OK rule only fires when ReportAll is set.
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestNoModuleJs(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := &analysis.Pass{
		RootDir: filepath.Join("./"),
		ResultOf: map[*analysis.Analyzer]interface{}{
			modulejs.Analyzer: map[string][]byte{},
		},
		Report: interceptor.ReportInterceptor(),
	}

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestPropTypes(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`MyComponent.propTypes={name:PropTypes.string}`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "react-19-prop-types", interceptor.Diagnostics[0].Name)
	require.Equal(t, analysis.Warning, interceptor.Diagnostics[0].Severity)
}

func TestDefaultProps(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`MyComponent.defaultProps={name:"default"}`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "react-19-prop-types", interceptor.Diagnostics[0].Name)
}

func TestContextTypes(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`MyComponent.contextTypes={theme:PropTypes.object}`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "react-19-legacy-context", interceptor.Diagnostics[0].Name)
}

func TestChildContextTypes(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`MyComponent.childContextTypes={theme:PropTypes.object}`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "react-19-legacy-context", interceptor.Diagnostics[0].Name)
}

func TestGetChildContext(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`getChildContext(){return{theme:this.state.theme}}`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "react-19-legacy-context", interceptor.Diagnostics[0].Name)
}

func TestStringRefsDoubleQuote(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`<input ref:"myInput"/>`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "react-19-string-refs", interceptor.Diagnostics[0].Name)
}

func TestStringRefsSingleQuote(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`<input ref:'myInput'/>`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "react-19-string-refs", interceptor.Diagnostics[0].Name)
}

func TestStringRefsNearMiss(t *testing.T) {
	// ref: without quotes around the value should not trigger
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`ref:someVariable`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestCreateFactory(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`var el=React.createFactory(MyComponent)`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "react-19-create-factory", interceptor.Diagnostics[0].Name)
}

func TestFindDOMNode(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`var node=ReactDOM.findDOMNode(this)`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "react-19-find-dom-node", interceptor.Diagnostics[0].Name)
}

func TestReactDOMRender(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`ReactDOM.render(<App/>,document.getElementById("root"))`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "react-19-legacy-render", interceptor.Diagnostics[0].Name)
}

func TestUnmountComponentAtNode(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`ReactDOM.unmountComponentAtNode(container)`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "react-19-legacy-render", interceptor.Diagnostics[0].Name)
}

func TestLegacyRenderNearMiss(t *testing.T) {
	// .render( without the ReactDOM. prefix should not trigger the legacy-render rule
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`component.render(props)`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 0)
}

func TestSecretInternals(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`React.__SECRET_INTERNALS_DO_NOT_USE_OR_YOU_WILL_BE_FIRED.ReactCurrentOwner`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "react-19-secret-internals", interceptor.Diagnostics[0].Name)
}

func TestMultipleIssues(t *testing.T) {
	// A bundle that hits several distinct rules should produce one diagnostic per rule.
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(
			`MyComponent.propTypes={name:PropTypes.string}` +
				`ReactDOM.render(<App/>,document.getElementById("root"))` +
				`var node=ReactDOM.findDOMNode(this)`,
		),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 3)

	names := make([]string, 0, 3)
	for _, d := range interceptor.Diagnostics {
		names = append(names, d.Name)
	}
	require.Contains(t, names, "react-19-prop-types")
	require.Contains(t, names, "react-19-legacy-render")
	require.Contains(t, names, "react-19-find-dom-node")
}

func TestDetailContainsUpgradeGuideLink(t *testing.T) {
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`MyComponent.propTypes={}`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Contains(t, interceptor.Diagnostics[0].Detail, react19UpgradeGuide)
}

func TestEachRuleReportedOnceEvenWithMultipleMatches(t *testing.T) {
	// Both .propTypes= and .defaultProps= match the same rule; only one diagnostic should be emitted.
	var interceptor testpassinterceptor.TestPassInterceptor
	pass := newPass(&interceptor, map[string][]byte{
		"module.js": []byte(`MyComponent.propTypes={} MyComponent.defaultProps={}`),
	})

	_, err := Analyzer.Run(pass)
	require.NoError(t, err)
	require.Len(t, interceptor.Diagnostics, 1)
	require.Equal(t, "react-19-prop-types", interceptor.Diagnostics[0].Name)
}
