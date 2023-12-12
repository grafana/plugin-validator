package sourcemap

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSourceMapFromBytes(t *testing.T) {
	tcs := []struct {
		desc              string
		json              string
		expectedError     string
		expectedSourceMap *SourceMap
	}{
		{
			desc: "empty sources",
			json: `{
				"version": 3,
				"sourcesContent": [
					"a",
					"b",
					"c"
				]
			}`,
			expectedError: "generated source code map requires the original source file paths to be present in the sources property",
		},
		{
			desc: "empty sourcesContent",
			json: `{
				"version": 3,
				"sources": [
					"a",
					"b",
					"c"
				]
			}`,
			expectedError: "generated source code map requires the original source code to be present in the sourcesContent property",
		},
		{
			desc: "source and sourcesContent mismatch",
			json: `{
				"version": 3,
				"sources": [
					"a",
					"b",
					"c"
				],
				"sourcesContent": [
					"d",
					"e"
				]
			}`,
			expectedError: "generated source code map requires the number of original source file paths (sources) to match with the number of original source code (sourcesContent)",
		},
		{
			desc: "valid",
			json: `{
				"version": 3,
				"sources": [
					"components/test.tsx",
					"external abc",
					"webpack/abc",
					"../node_modules/abc",
					"./node_modules/abc",
					"components/test2.tsx"
				],
				"sourcesContent": [
					"abc",
					"",
					"",
					"",
					"",
					"def"
				]
			}`,
			expectedSourceMap: &SourceMap{
				Version: 3,
				Sources: map[string]string{
					"components/test.tsx":  "abc",
					"components/test2.tsx": "def",
				},
			},
		},
		{
			desc: "valid",
			json: `{
				"version": 3,
				"sources": [
					"components/test.tsx",
					"external abc",
					"webpack/abc",
					"../node_modules/abc",
					"./node_modules/abc",
					"components/test2.tsx",
					"./test.svg",
					"./img/ sync ^\\.\/.*\\.svg$"
				],
				"sourcesContent": [
					"abc",
					"",
					"",
					"",
					"",
					"def",
					"",
					""
				]
			}`,
			expectedSourceMap: &SourceMap{
				Version: 3,
				Sources: map[string]string{
					"components/test.tsx":  "abc",
					"components/test2.tsx": "def",
				},
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			pluginID := "test-anyorg-panel"
			replaceRegex, err := regexp.Compile(fmt.Sprintf("^webpack://%s/*|^webpack:/*", pluginID))
			require.NoError(t, err)
			sourceMap, err := ParseSourceMapFromBytes(replaceRegex, []byte(tc.json))
			if tc.expectedError != "" {
				require.Equal(t, tc.expectedError, err.Error())
			} else {
				require.NoError(t, err)
			}

			if tc.expectedSourceMap != nil {
				require.Equal(t, tc.expectedSourceMap, sourceMap)
			} else {
				require.Nil(t, tc.expectedSourceMap)
			}
		})
	}
}
