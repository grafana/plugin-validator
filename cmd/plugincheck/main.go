package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/grafana/plugin-validator/pkg/grafana"
	"github.com/grafana/plugin-validator/pkg/plugin"
)

func main() {
	var (
		pluginURLFlag  = flag.String("url", "", "URL to the plugin")
		schemaPathFlag = flag.String("schema", "./config/plugin.schema.json", "Path to the JSON Schema to validate against.")
		strictFlag     = flag.Bool("strict", false, "If set, plugincheck returns non-zero exit code for warnings")
	)

	flag.Parse()

	if *pluginURLFlag == "" {
		fmt.Fprintln(os.Stderr, "missing plugin url")
		os.Exit(1)
	}

	client := grafana.NewClient()

	_, result, err := plugin.Check(*pluginURLFlag, *schemaPathFlag, client)
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
