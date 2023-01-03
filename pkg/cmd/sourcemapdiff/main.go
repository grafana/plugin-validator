package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/grafana/plugin-validator/pkg/cmd/sourcemapdiff/difftool"
)

var options = map[string]*string{
	"archiveUri":    flag.String("archiveUri", "", "The URI of the plugin archive. This can be a local file path (zip or folder) or a URL. If it's a URL, it must be a zip file."),
	"sourceCodeUri": flag.String("sourceCodeUri", "", "The URI of the source code. This can be a local file path (zip or folder) or a URL. If it's a URL, it must be a git repository or a zip file."),
}

func main() {
	flag.Parse()

	// resolve the pluginPath from options
	pluginPath, archiveCleanup, err := getLocalPathFromArchiveOption(*options["archiveUri"])
	defer archiveCleanup()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// resolve the sourceCodePath from options
	sourceCodePath, sourceCodeCleanup, err := getLocalPathFromSourceCodeOption(*options["sourceCodeUri"])
	defer sourceCodeCleanup()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// extract the source map into a temporal folder
	sourceCodeMapPath := filepath.Join(pluginPath, "module.js.map")

	// compare the source map with the source code
	report, err := difftool.CompareSourceMapToSourceCode(sourceCodeMapPath, filepath.Join(sourceCodePath, "src"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println(report.GeneratePrintableReport())

}
