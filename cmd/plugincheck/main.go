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
		strictFlag = flag.Bool("strict", false, "If set, plugincheck returns non-zero exit code for warnings")
	)

	flag.Parse()

	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "missing plugin url")
		os.Exit(1)
	}

	pluginURL := os.Args[1]

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

	_, result, err := plugin.Check(pluginURL, schemaFile.Name(), client)
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
