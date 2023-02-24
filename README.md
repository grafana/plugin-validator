# Grafana Plugin Validator

[![License](https://img.shields.io/github/license/grafana/plugin-validator)](LICENSE)

A tool for validating community plugins for publishing to Grafana.com.

The tool expects path to either a remote or a local ZIP archive.

## Install

```SHELL
cd pkg/cmd/plugincheck2
go install
```

## Run

Typically running the checker with default settings is the easiest method to see if there are issues with a plugin.

```SHELL
plugincheck2 -config config/default.yaml https://github.com/marcusolsson/grafana-jsonapi-datasource/releases/download/v0.6.0/marcusolsson-json-datasource-0.6.0.zip
```

To wrap the output with another tool (like the validator ui), running with the `terse-json.yaml` config can be used.

```SHELL
plugincheck2 -config config/terse-json.yaml https://github.com/marcusolsson/grafana-jsonapi-datasource/releases/download/v0.6.0/marcusolsson-json-datasource-0.6.0.zip
```

Verbose json output is available to show all checks made, with status for each.

```SHELL
plugincheck2 -config config/verbose-json.yaml https://github.com/marcusolsson/grafana-jsonapi-datasource/releases/download/v0.6.0/marcusolsson-json-datasource-0.6.0.zip
```

## Configuration

You must pass a configuration file to the validator with the `-config` option. Several configuraton examples are available to use here <https://github.com/grafana/plugin-validator/tree/main/config>

### Enabling and disabling analyzers via config

If you wish to disable an specific check (analyzer) you can define this in your configuration file adding an `analyzers` section and specyfing which analyzer or analyzer rules to enable and disable.

E.g.: disable the `version` analyzer

```yaml
global:
  enabled: true
  jsonOutput: false
  reportAll: false

analyzers:
  version:
    enabled: false
```

You can also disable specific rules or change their severity level:

```yaml
global:
  enabled: true
  jsonOutput: false
  reportAll: false

analyzers:
  readme:
    rules:
      missing-readme:
        enabled: true
        severity: warning
```

Severity levels could be: `error`, `warning`, `ok`

Please notice that Grafanalabs uses its own configuration for plugins submissions and your own config file can't change these rules.

## Analyzers

The plugincheck tool runs a series of analyzers to ensure submitted plugins are following best practices, and speed up the process of approving a plugin for publishing.

Currently there are 20 different types of checks being performed, and are described below.

### Archive Structure

Ensures the contents of the zip file have the expected layout.

### Archive Name

The name of the archive should be correctly formatted.

### Binary Permissions

For datasources and apps with binaries, this ensures the plugin can run when extracted on a system.

### Broken Links

Detects if any url does not resolve to a valid location.

### HTML in Readme

Detects if there are any html tags used in the README.md, as they will not render in the marketplace.

### Developer Jargon

Generally discourage use of code jargon in the documentation.

### Legacy Platform

### License Type

Ensures the license type specified is allowed.

### Manifest (Signing)

When a plugin is signed, the zip file will contain a signed MANIFEST.txt file.

### Metadata Paths and Validity

Ensures all paths are valid and images referenced exist.

### module.js (exists)

All plugins require a module.js to be loaded.

### Organization exists

Verifies the org specified in the plugin id exists.

### Plugin Name formatting

Validates the plugin id used conforms to our naming convention.

### Readme (exists)

Ensures a README.md file exists within the zip file.

### Restrictive Dependency

Specifies a valid range of Grafana that works with this version of the plugin.

### Screenshots

Screenshots are specified in plugin.json that will be used in the marketplace.

### Signature

Ensures the plugin has a valid signature.

### Source Code (NEW!)

The source code URI matches the released code. A comparison is made between the zip file and the source code to ensure what is released matches the repo associated with it.

### Unique README.md

Ensures the plugin does not re-use the template from the create-plugin tool.

### No Tracking Scripts

Detects if there are any known tracking scripts, which are not allowed.

### Type Suffix (panel/app/datasource)

Ensures the plugin has a valid type specified.

### Version

Ensures the version submitted is newer than the currently published plugin. If this is a new/unpublished plugin, this is skipped.

### Vulnerability Scanner

This analyzer leverages the OSV Scanner (<https://github.com/google/osv-scanner>) to detect critical vulnerabilities in go modules and yarn lock files.

Any critical vulnerability will cause the validation to fail and prevent a plugin from being published.

Source code must be provided for this analyzer to execute, and osv-scanner needs to be in your PATH for it to run.

#### Running this Analyzer

Default Usage:

```SHELL
plugincheck2 -config config/default.yaml -sourceCodeUri https://github.com/briangann/grafana-gauge-panel/archive/refs/tags/v0.0.9.zip https://github.com/briangann/grafana-gauge-panel/releases/download/v0.0.9/briangann-gauge-panel-0.0.9.zip
```

Example default output:

```TEXT
warning: README.md: possible broken link: https://www.d3js.org (404 Not Found)
detail: README.md might contain broken links. Check that all links are valid and publicly accesible.
warning: README.md contains developer jargon: (yarn)
detail: Move any developer and contributor documentation to a separate file and link to it from the README.md. For example, CONTRIBUTING.md, DEVELOPMENT.md, etc.
error: osv-scanner detected a critical severity issue
detail: SEVERITY: CRITICAL in package immer, vulnerable to CVE-2021-23436
error: osv-scanner detected a critical severity issue
detail: SEVERITY: CRITICAL in package json-schema, vulnerable to CVE-2021-3918
error: osv-scanner detected a critical severity issue
detail: SEVERITY: CRITICAL in package loader-utils, vulnerable to CVE-2022-37601
error: osv-scanner detected a critical severity issue
detail: SEVERITY: CRITICAL in package minimist, vulnerable to CVE-2021-44906
error: osv-scanner detected a critical severity issue
detail: SEVERITY: CRITICAL in package shell-quote, vulnerable to CVE-2021-42740
error: osv-scanner detected a critical severity issue
detail: SEVERITY: CRITICAL in package simple-git, vulnerable to CVE-2022-25860
error: osv-scanner detected critical severity issues
detail: osv-scanner detected 6 unique critical severity issues for lockfile: /var/folders/84/yw3k27_j0d79r_myzgjgx1980000gn/T/validator1049475772/yarn.lock
error: Plugin version 0.0.9 is invalid.
detail: The submitted plugin version 0.0.9 is not greater than the latest published version 0.0.9 on grafana.com.
```

Terse JSON Output:

```SHELL
plugincheck2 -config config/terse-json.yaml -sourceCodeUri https://github.com/briangann/grafana-gauge-panel/archive/refs/tags/v0.0.9.zip https://github.com/briangann/grafana-gauge-panel/releases/download/v0.0.9/briangann-gauge-panel-0.0.9.zip
```

JSON:

```JSON
{
  "id": "briangann-gauge-panel",
  "version": "0.0.9",
  "plugin-validator": {
    "brokenlinks": [
      {
        "Severity": "warning",
        "Title": "README.md: possible broken link: https://www.d3js.org (404 Not Found)",
        "Detail": "README.md might contain broken links. Check that all links are valid and publicly accesible.",
        "Name": "broken-link"
      }
    ],
    "jargon": [
      {
        "Severity": "warning",
        "Title": "README.md contains developer jargon: (yarn)",
        "Detail": "Move any developer and contributor documentation to a separate file and link to it from the README.md. For example, CONTRIBUTING.md, DEVELOPMENT.md, etc.",
        "Name": "developer-jargon"
      }
    ],
    "osv-scanner": [
      {
        "Severity": "error",
        "Title": "osv-scanner detected a critical severity issue",
        "Detail": "SEVERITY: CRITICAL in package immer, vulnerable to CVE-2021-23436",
        "Name": "osvscanner-critical-severity-vulnerabilities-detected"
      },
      {
        "Severity": "error",
        "Title": "osv-scanner detected a critical severity issue",
        "Detail": "SEVERITY: CRITICAL in package json-schema, vulnerable to CVE-2021-3918",
        "Name": "osvscanner-critical-severity-vulnerabilities-detected"
      },
      {
        "Severity": "error",
        "Title": "osv-scanner detected a critical severity issue",
        "Detail": "SEVERITY: CRITICAL in package loader-utils, vulnerable to CVE-2022-37601",
        "Name": "osvscanner-critical-severity-vulnerabilities-detected"
      },
      {
        "Severity": "error",
        "Title": "osv-scanner detected a critical severity issue",
        "Detail": "SEVERITY: CRITICAL in package minimist, vulnerable to CVE-2021-44906",
        "Name": "osvscanner-critical-severity-vulnerabilities-detected"
      },
      {
        "Severity": "error",
        "Title": "osv-scanner detected a critical severity issue",
        "Detail": "SEVERITY: CRITICAL in package shell-quote, vulnerable to CVE-2021-42740",
        "Name": "osvscanner-critical-severity-vulnerabilities-detected"
      },
      {
        "Severity": "error",
        "Title": "osv-scanner detected a critical severity issue",
        "Detail": "SEVERITY: CRITICAL in package simple-git, vulnerable to CVE-2022-25860",
        "Name": "osvscanner-critical-severity-vulnerabilities-detected"
      },
      {
        "Severity": "error",
        "Title": "osv-scanner detected critical severity issues",
        "Detail": "osv-scanner detected 6 unique critical severity issues for lockfile: /var/folders/84/yw3k27_j0d79r_myzgjgx1980000gn/T/validator1150112313/yarn.lock",
        "Name": "osvscanner-critical-severity-vulnerabilities-detected"
      }
    ],
    "version": [
      {
        "Severity": "error",
        "Title": "Plugin version 0.0.9 is invalid.",
        "Detail": "The submitted plugin version 0.0.9 is not greater than the latest published version 0.0.9 on grafana.com.",
        "Name": "wrong-plugin-version"
      }
    ]
  }
}
```

Verbose

```SHELL
plugincheck2 -config config/verbose-json.yaml -sourceCodeUri https://github.com/briangann/grafana-gauge-panel/archive/refs/tags/v0.0.9.zip https://github.com/briangann/grafana-gauge-panel/releases/download/v0.0.9/briangann-gauge-panel-0.0.9.zip
```

## License

Grafana Plugin Validator is distributed under the [Apache 2.0 License](https://github.com/grafana/plugin-validator/blob/master/LICENSE).
