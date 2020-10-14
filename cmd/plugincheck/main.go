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

func main() {
	var (
		pluginURLFlag  = flag.String("url", "", "URL to the plugin")
		schemaPathFlag = flag.String("schema", "", "Deprecated. Path to the JSON Schema to validate against.")
		strictFlag     = flag.Bool("strict", false, "If set, plugincheck returns non-zero exit code for warnings")
	)

	flag.Parse()

	if *schemaPathFlag != "" {
		fmt.Println("Warning: The schema flag has been deprecated. plugincheck now downloads the schema from the Grafana repository.")
	}

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

	if *pluginURLFlag == "" {
		fmt.Fprintln(os.Stderr, "missing plugin url")
		os.Exit(1)
	}

	client := grafana.NewClient()

	_, result, err := plugin.Check(*pluginURLFlag, schemaFile.Name(), client)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for _, e := range result {
		b, err := json.MarshalIndent(e, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		os.Stdout.Write(b)
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
