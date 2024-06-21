package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/grafana/plugin-validator/pkg/prettyprint"
	"github.com/r3labs/diff"
	"github.com/stretchr/testify/assert"
)

type Issue struct {
	Severity string `json:"severity"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Name     string `json:"name"`
}

type JsonReport struct {
	Id              string             `json:"id"`
	Version         string             `json:"version"`
	PluginValidator map[string][]Issue `json:"plugin-validator"`
}

func TestIntegration(t *testing.T) {
	basePath := "./testdata"
	binary := filepath.Join(
		"..",
		"..",
		"..",
		"bin",
		runtime.GOOS+"_"+runtime.GOARCH,
		"plugincheck2",
	)
	env := []string{
		"DEBUG=0",
		"OPENAI_API_KEY=",
		"GEMINI_API_KEY=",
	}
	var files = map[string]JsonReport{
		"grafana-clock-panel-2.1.5.any.zip": {
			Id:      "grafana-clock-panel",
			Version: "2.1.5",
			PluginValidator: map[string][]Issue{
				"jargon": {
					{
						Severity: "warning",
						Title:    "README.md contains developer jargon: (yarn)",
						Detail:   "Move any developer and contributor documentation to a separate file and link to it from the README.md. For example, CONTRIBUTING.md, DEVELOPMENT.md, etc.",
						Name:     "developer-jargon",
					},
				},
			},
		},
		"alexanderzobnin-zabbix-app-4.4.9.linux_amd64.zip": {
			Id:      "alexanderzobnin-zabbix-app",
			Version: "4.4.9",
			PluginValidator: map[string][]Issue{
				"includesnested": {
					{
						Severity: "error",
						Title:    "Nested plugin datasource/plugin.json is not declared parent plugin.json",
						Detail:   "Found a plugin datasource/plugin.json nested inside your archive but not declared in plugin.json. Make sure to declare the type and path of the nested plugin",
						Name:     "nested-plugins-not-declared",
					},
					{
						Severity: "error",
						Title:    "Nested plugin panel-triggers/plugin.json is not declared parent plugin.json",
						Detail:   "Found a plugin panel-triggers/plugin.json nested inside your archive but not declared in plugin.json. Make sure to declare the type and path of the nested plugin",
						Name:     "nested-plugins-not-declared",
					},
				},
			},
		},
		"yesoreyeram-infinity-datasource-2.6.3.linux_amd64.zip": {
			Id:              "yesoreyeram-infinity-datasource",
			Version:         "2.6.3",
			PluginValidator: map[string][]Issue{},
		},
		"invalid.zip": {
			Id:      "invalid-panel",
			Version: "1.0.0",
			PluginValidator: map[string][]Issue{
				"archive": {
					{
						Severity: "error",
						Title:    "Archive contains more than one directory",
						Detail:   "Archive should contain only one directory named after plugin id. Found 7 directories",
						Name:     "more-than-one-dir",
					},
					{
						Severity: "error",
						Title:    "Plugin archive is improperly structured",
						Detail:   "",
						Name:     "zip-invalid",
					},
				},
			},
		},
		"invalid2.zip": {
			Id:      "invalid-panel",
			Version: "1.0.0",
			PluginValidator: map[string][]Issue{
				"archivename": {
					{
						Severity: "error",
						Title:    "Archive should contain a directory named invalid-panel",
						Detail:   "The plugin archive file should contain a directory named after the plugin ID. This directory should contain the plugin's dist files.",
						Name:     "no-ident-root-dir",
					},
				},
				"license": {
					{
						Severity: "error",
						Title:    "LICENSE file could not be parsed.",
						Detail:   "Could not parse the license file inside the plugin archive. Please make sure to include a valid license in your LICENSE file in your archive.",
						Name:     "license-not-provided",
					},
				},
				"manifest": {
					{
						Severity: "warning",
						Title:    "unsigned plugin",
						Detail:   "This is a new (unpublished) plugin. This is expected during the initial review process. Please allow the review to continue, and a member of our team will inform you when your plugin can be signed.",
						Name:     "unsigned-plugin",
					},
				},
				"metadatapaths": {
					{
						Severity: "error",
						Title:    "plugin.json: small logo path doesn't exists: img/logo.svg",
						Detail:   "Refer only existing files. Make sure the files referred in plugin.json are included in the archive.",
						Name:     "path-not-exists",
					},
					{
						Severity: "error",
						Title:    "plugin.json: large logo path doesn't exists: img/logo.svg",
						Detail:   "Refer only existing files. Make sure the files referred in plugin.json are included in the archive.",
						Name:     "path-not-exists",
					},
				},
				"metadatavalid": {
					{
						Severity: "warning",
						Title:    "plugin.json: dependencies: grafanaDependency is required",
						Detail:   "The plugin.json file is not following the schema. Please refer to the documentation for more information. https://grafana.com/docs/grafana/latest/developers/plugins/metadata/",
						Name:     "invalid-metadata",
					},
				},
				"readme": {
					{
						Severity: "error",
						Title:    "README.md is empty",
						Detail:   "A README.md file is required for plugins. The contents of the file will be displayed in the Plugin catalog.",
						Name:     "missing-readme",
					},
				},
				"screenshots": {
					{
						Severity: "warning",
						Title:    "plugin.json: should include screenshots for the Plugin catalog",
						Detail:   "Screenshots are displayed in the Plugin catalog. Please add at least one screenshot to your plugin.json.",
						Name:     "screenshots",
					},
				},
			},
		},
	}

	configFile := filepath.Join(basePath, "integration-tests.yaml")

	t.Logf("Running integration tests. Total: %d\n", len(files))
	for currentFile := range files {
		t.Run(currentFile, func(t *testing.T) {
			file := currentFile
			// Allows the test case to run in parallel with other ones
			t.Parallel()

			t.Logf("Running %s", file)

			command := fmt.Sprintf(
				"%s -config %s %s",
				binary,
				configFile,
				filepath.Join(basePath, file),
			)
			t.Logf("Running command: %s\n", command)
			cmd := exec.Command("sh", "-c", command)
			var outb, errb bytes.Buffer
			cmd.Stdout = &outb
			cmd.Stderr = &errb
			cmd.Env = append(os.Environ(), env...)
			err := cmd.Run()
			if err != nil && len(outb.String()) == 0 {
				prettyprint.Print(errb.String())
				assert.NoError(t, err)
			}

			// marshall the output into a JsonReport
			var report JsonReport
			err = json.Unmarshal(outb.Bytes(), &report)
			assert.NoError(t, err)

			changelog, err := diff.Diff(files[file], report)
			assert.NoError(t, err)

			if len(changelog) > 0 {
				t.Logf("Changes found: %d\n", len(changelog))
				prettyJson, _ := json.MarshalIndent(changelog, "", "\t")
				fmt.Println(string(prettyJson))
			}
			assert.Len(t, changelog, 0)
		})
	}
}
