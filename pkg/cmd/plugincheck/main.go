package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/grafana/plugin-validator/pkg/grafana"
	"github.com/grafana/plugin-validator/pkg/plugin"
)

// Deprecated: plugincheck V1 is deprecated and will be removed in a future release.
// Use plugincheck2 instead. See https://github.com/grafana/plugin-validator
func main() {

	fmt.Println(`
     888                                               888                888
     888                                               888                888
     888                                               888                888
 .d88888 .d88b. 88888b. 888d888 .d88b.  .d8888b 8888b. 888888 .d88b.  .d88888
d88" 888d8P  Y8b888 "88b888P"  d8P  Y8bd88P"       "88b888   d8P  Y8bd88" 888
888  88888888888888  888888    88888888888     .d888888888   88888888888  888
Y88b 888Y8b.    888 d88P888    Y8b.    Y88b.   888  888Y88b. Y8b.    Y88b 888
 "Y88888 "Y8888 88888P" 888     "Y8888  "Y8888P"Y888888 "Y888 "Y8888  "Y88888
                888
                888
                888

	**plugincheck V1 is deprecated and you should not use it.

	Use V2 instead: plugincheck2

	To install install it see README https://github.com/grafana/plugin-validator`)
	fmt.Println()
	var (
		strictFlag  = flag.Bool("strict", false, "If set, plugincheck returns non-zero exit code for warnings")
		privateFlag = flag.Bool("private", false, "If set, plugincheck reports private signature check error as warning")
	)

	flag.Parse()

	if len(flag.Args()) < 1 {
		fmt.Fprintln(os.Stderr, "missing plugin url")
		os.Exit(1)
	}

	pluginURL := flag.Arg(0)

	schemaFile, err := ioutil.TempFile("", "plugin_*.schema.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "couldn't create schema file")
		os.Exit(1)
	}
	defer os.Remove(schemaFile.Name())

	resp, err := http.Get("https://raw.githubusercontent.com/grafana/grafana/master/docs/sources/developers/plugins/plugin.schema.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, "couldn't download plugin schema")
		os.Exit(1)
	}
	defer resp.Body.Close()

	_, err = io.Copy(schemaFile, resp.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "couldn't download plugin schema")
		os.Exit(1)
	}

	client := grafana.NewClient()

	_, result, err := plugin.Check(pluginURL, schemaFile.Name(), *privateFlag, client)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, e := range result {
		enc := json.NewEncoder(os.Stdout)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")

		err := enc.Encode(e)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if len(result) > 0 {
		if *strictFlag {
			os.Exit(1)
		}

		for _, res := range result {
			if res.Severity == "error" {
				os.Exit(1)
			}
		}
	}
}
