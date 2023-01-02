package main

import (
	"flag"
	"fmt"
	"os"
)

var options = map[string]*string{
	"archiveUri":    flag.String("archiveUri", "", "The URI of the plugin archive. This can be a local file path (zip or folder) or a URL. If it's a URL, it must be a zip file."),
	"sourceCodeUri": flag.String("sourceCodeUri", "", "The URI of the source code. This can be a local file path (zip or folder) or a URL. If it's a URL, it must be a git repository or a zip file."),
}

func main() {
	flag.Parse()
	pluginPath, archiveCleanup, err := getLocalPathFromArchiveOption(*options["archiveUri"])
	defer archiveCleanup()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	sourceCodePath, sourceCodeCleanup, err := getLocalPathFromSourceCodeOption(*options["sourceCodeUri"])
	defer sourceCodeCleanup()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Println("--->")
	fmt.Println("plugin path code", pluginPath)
	fmt.Println("source path code", sourceCodePath)

}
