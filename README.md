# Grafana Plugin Validator

[![License](https://img.shields.io/github/license/grafana/plugin-validator)](LICENSE)
[![Drone](https://drone.grafana.net/api/badges/grafana/plugin-validator/status.svg)](https://drone.grafana.net/grafana/plugin-validator)
[![Go Report Card](https://goreportcard.com/badge/github.com/grafana/plugin-validator)](https://goreportcard.com/report/github.com/grafana/plugin-validator)

This tool helps speed up the process of publishing plugins to [Grafana.com](https://grafana.com/grafana/plugins/). It runs a series of [analyzers](#analyzers) to ensure plugins are following best practices, checking for security and structural issues, as well as specific requirements related to publishing. A general overview of these requirements can be found here: <https://grafana.com/docs/grafana/latest/developers/plugins/publishing-and-signing-criteria/>.

It requires a path to a remote or local ZIP archive of the plugin to be specified, for example:

- **Remote**: `https://github.com/grafana/clock-panel/releases/download/v2.1.2/grafana-clock-panel-2.1.2.zip`
- **Local**: `file://Users/me/Downloads/grafana-clock-panel-2.1.2.zip`

You can _additionally_ provide a link to the source code for the project with `-sourceCodeUri` to enable additional analyzers such as the Vulnerability Scan.

## Installation and usage

Ensure that your version of Go matches the one specified in the `go.mod` file to avoid compatibility issues

### Docker (recommended)

It is easiest to run the tool using the Docker image as it contains all the [security scanning tools](#security-tools) needed for the full set of analyzers - so you don't need to have these additional tools installed on your system.

```SHELL
docker run --pull=always grafana/plugin-validator-cli [options] [http://yourdomain/plugin_archive.zip]
```

#### Example 1 (basic)

```SHELL
docker run --pull=always grafana/plugin-validator-cli https://github.com/grafana/clock-panel/releases/download/v2.1.2/grafana-clock-panel-2.1.2.zip
```

#### Example 2 (specifying source code location)

```SHELL
docker run --pull=always grafana/plugin-validator-cli -sourceCodeUri https://github.com/grafana/clock-panel/tree/v2.1.2 https://github.com/grafana/clock-panel/releases/download/v2.1.2/grafana-clock-panel-2.1.2.zip
```

#### Using a local archive file with Docker

To run the tool with a local archive you will need to mount it as a docker volume. Here's an example:

```SHELL
docker run --pull=always -v /path/to/plugin_archive.zip:/archive.zip grafana/plugin-validator-cli /archive.zip
```

> [!NOTE]
> If using relative paths your path must start with `./`

#### Using a local archive file and local source code

```SHELL
docker run --pull=always -v /path/to/plugin_archive.zip:/archive.zip -v /path/to/source_code:/source_code grafana/plugin-validator-cli -sourceCodeUri /source_code /archive.zip
```

> [!NOTE]
> If using relative paths your path must start with `./`

### NPX

```SHELL
npx -y @grafana/plugin-validator@latest -sourceCodeUri [options] [path/to/plugin_archive.zip]
```

### Locally

First you must compile and install it:

```SHELL
git clone git@github.com:grafana/plugin-validator.git
cd plugin-validator/pkg/cmd/plugincheck2
go install
```

Then you can run the utility:

```SHELL
plugincheck2 -sourceCodeUri [source_code_location/] [plugin_archive.zip]
```

### Generating local files For validation

You must create a `.zip` archive containing the `dist/` directory but named as your plugin ID:

```SHELL
PLUGIN_ID=$(grep '"id"' < src/plugin.json | sed -E 's/.*"id" *: *"(.*)".*/\1/')
cp -r dist "${PLUGIN_ID}"
zip -qr "${PLUGIN_ID}.zip" "${PLUGIN_ID}"
npx @grafana/plugin-validator@latest -sourceCodeUri file://. "${PLUGIN_ID}.zip"
```

You can optionally remove the files that were generated:

```SHELL
rm -r "${PLUGIN_ID}" "${PLUGIN_ID}.zip"
```

## Options

Additional options can be passed to the tool:

```BASH
❯ plugincheck2 -help
Usage plugincheck2:
  -config string (optional)
        Path to configuration file
  -sourceCodeUri string (optional)
        URI to the source code of the plugin. If set, the source code will be downloaded and analyzed. This can be a ZIP file URL, a URL to git repository or a local file (starting with `file://`)
  -strict (optional)
        If set, plugincheck returns non-zero exit code for warnings
  -checksum string (optional)
        If set, the checksum of the plugin archive will be checked against this value. MD5 and SHA256 are supported.
  -analyzer string (optional)
        If set, only an specific analyzer and it's dependencies will run.
  -severity string (optional)
        If used, it will set the severity of the analyzer (it has the highest priority).

```

### Using a configuration file

You can pass a configuration YAML file to the validator with the `-config` option. Several configuration examples are available to use here: <https://github.com/grafana/plugin-validator/tree/main/config>.

#### Enabling and disabling analyzers via config

If you want to disable an specific check (analyzer) you can define this in your [configuration file](#using-a-configuration-file), adding an `analyzers` section, and specifying which analyzer or analyzer rules to enable and disable.

For example, disable the `version` analyzer:

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

Severity levels could be: `error`, `warning`, or `ok`.

> Note: Grafana Labs enforces its own configuration for plugins submissions and your own config file can't change these rules.

### Source code

You can specify the location of the plugin source code to the validator with the `-sourceCodeUri` option. Doing so allows for additional [analyzers](#analyzers) to be run and for a more complete scan.

### Supported remote Git services

The following **public** Git services are supported:

- GitHub
- GitLab
- Bitbucket

Private repositories are not currently supported.

Make sure to include the `ref` (branch or tag) of the corresponding source code.

For example: you are validating version `v2.1.2` and your project is in GitHub. Make sure you create a corresponding tag or branch and use the URL `https://github.com/grafana/clock-panel/tree/v2.1.2`.

## Debug mode

You can run the validator in debug mode to get more information about the running checks and possible errors.

Docker:

```SHELL
docker run --pull=always -e DEBUG=1 grafana/plugin-validator-cli -sourceCodeUri https://github.com/grafana/clock-panel/tree/v2.1.2 https://github.com/grafana/clock-panel/releases/download/v2.1.2/grafana-clock-panel-2.1.2.zip
```

NPX:

```SHELL
DEBUG=1 npx -y @grafana/plugin-validator@latest -sourceCodeUri https://github.com/grafana/clock-panel/tree/v2.1.2 https://github.com/grafana/clock-panel/releases/download/v2.1.2/grafana-clock-panel-2.1.2.zip
```

Locally:

```SHELL
DEBUG=1 plugincheck2 -sourceCodeUri https://github.com/grafana/clock-panel/tree/v2.1.2 https://github.com/grafana/clock-panel/releases/download/v2.1.2/grafana-clock-panel-2.1.2.zip
```

## Security tools

This validator makes uses of the following open source security tools:

- [osv-scanner](https://github.com/google/osv-scanner)
- [semgrep](https://github.com/returntocorp/semgrep)
- [gosec](https://github.com/securego/gosec)

If you run the validator locally or via NPX you can benefit from installing these tools in your system to make them part of your validation checks.

---

## Analyzers

The tool runs a series of analyzers to ensure submitted plugins are following best practices, and speed up the process of approving a plugin for publishing, detailed in the table below. The _Analyzer_ column includes the name required for altering the behavior of a given check in a [configuration file](#using-a-configuration-file). The _Dependencies_ column specifies whether the analyzer requires the source code for the plugin to be provided with `sourceCodeUri` or for any additional [security scanning tools](#security-tools) to be present.

| Analyzer                                          | Description                                                                                                                       | Dependencies                                                          |
| ------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------- |
| Archive Structure / `archive`                     | Ensures the contents of the zip file have the expected layout.                                                                    | None                                                                  |
| Archive Name / `archivename`                      | The name of the archive should be correctly formatted.                                                                            | None                                                                  |
| Backend Binary / `backendbinary`                  | Validates the consistency between the existence of a binary file and plugin.json declarations for backend or alerting.            | None                                                                  |
| Backend Debug / `backenddebug`                    | Checks that the standalone debug files for backend plugins are not present.                                                       | None                                                                  |
| Binary Permissions / `binarypermissions`          | For datasources and apps with binaries, this ensures the plugin can run when extracted on a system.                               | None                                                                  |
| Broken Links / `brokenlinks`                      | Detects if any URL doesn't resolve to a valid location.                                                                           | None                                                                  |
| Checksum / `checksum`                             | Validates that the passed checksum (as a validator arg) is the one calculated from the archive file.                              | `checksum`                                                            |
| Code Rules / `coderules`                          | Checks for forbidden access to environment variables, file system or use of `syscall` module.                                     | [semgrep](https://github.com/returntocorp/semgrep), `sourceCodeUri`   |
| Discoverability / `discoverability`               | Warns about missing keywords and description that are used for plugin indexing in the catalog.                                    | None                                                                  |
| Go Manifest / `gomanifest`                        | Validates the build manifest.                                                                                                     | `sourceCodeUri`                                                       |
| Go Security Checker / `gosec`                     | Inspects source code for security problems by scanning the Go AST.                                                                | [gosec](https://github.com/securego/gosec), `sourceCodeUri`           |
| Nested includes metadata / `includesnested`       | Validates that nested plugins have the correct metadata.                                                                          |                                                                       |
| Developer Jargon / `jargon`                       | Generally discourages use of code jargon in the documentation.                                                                    | None                                                                  |
| JS Source Map / `jssourcemap`                     | Checks for required `module.js.map` file(s) in archive.                                                                           | `sourceCodeUri`                                                       |
| Legacy Grafana Toolkit usage / `legacybuilder`    | Detects the usage of the not longer supported Grafana Toolkit.                                                                    | None                                                                  |
| Legacy Platform / `legacyplatform`                | Detects use of Angular which is deprecated.                                                                                       | None                                                                  |
| License Type / `license`                          | Checks the declared license is one of: BSD, MIT, Apache 2.0, LGPL3, GPL3, AGPL3.                                                  | None                                                                  |
| Logos / `logos`                                   | Detects whether the plugin includes small and large logos to display in the plugin catalog.                                       | none                                                                  |
| Manifest (Signing) / `manifest`                   | When a plugin is signed, the zip file will contain a signed `MANIFEST.txt` file.                                                  | None                                                                  |
| Metadata Paths / `metadatapaths`                  | Ensures all paths are valid and images referenced exist.                                                                          | None                                                                  |
| Metadata Validity / `metadatavalid`               | Ensures metadata is valid and matches plugin schema.                                                                              | None                                                                  |
| module.js (exists) / `modulejs`                   | All plugins require a `module.js` to be loaded.                                                                                   | None                                                                  |
| Organization (exists) / `org`                     | Verifies the org specified in the plugin ID exists.                                                                               | None                                                                  |
| Vulnerability Scanner / `osv-scanner`             | Detects critical vulnerabilities in Go modules and yarn lock files.                                                               | [osv-scanner](https://github.com/google/osv-scanner), `sourceCodeUri` |
| Plugin Name formatting / `pluginname`             | Validates the plugin ID used conforms to our naming convention.                                                                   | None                                                                  |
| Published / `published`                           | Detects whether any version of this plugin exists in the Grafana plugin catalog currently.                                        | None                                                                  |
| Readme (exists) / `readme`                        | Ensures a `README.md` file exists within the zip file.                                                                            | None                                                                  |
| Restrictive Dependency / `restrictivedep`         | Specifies a valid range of Grafana versions that work with this version of the plugin.                                            | None                                                                  |
| Screenshots / `screenshots`                       | Screenshots are specified in `plugin.json` that will be used in the Grafana plugin catalog.                                       | None                                                                  |
| Signature / `signature`                           | Ensures the plugin has a valid signature.                                                                                         | None                                                                  |
| !Source Code / `sourcecode`                       | A comparison is made between the zip file and the source code to ensure what is released matches the repo associated with it.     | `sourceCodeUri`                                                       |
| Unique README.md / `templatereadme`               | Ensures the plugin doesn't re-use the template from the `create-plugin` tool.                                                     | None                                                                  |
| No Tracking Scripts / `trackingscripts`           | Detects if there are any known tracking scripts, which are not allowed.                                                           | None                                                                  |
| Type Suffix (panel/app/datasource) / `typesuffix` | Ensures the plugin has a valid type specified.                                                                                    | None                                                                  |
| Version / `version`                               | Ensures the version submitted is newer than the currently published plugin. If this is a new/unpublished plugin, this is skipped. | None                                                                  |
| Unsafe SVG / `unsafesvg`                          | Checks if any svg files are safe based on a whitelist of elements and attributes.                                                 | none                                                                  |

## Output

By default, the tool outputs results in plain text as shown below.

Default:

```TEXT
warning: README.md: possible broken link: https://www.d3js.org (404 Not Found)
detail: README.md might contain broken links. Check that all links are valid and publicly accessible.
warning: README.md contains developer jargon: (yarn)
detail: Move any developer and contributor documentation to a separate file and link to it from the README.md. For example, CONTRIBUTING.md, DEVELOPMENT.md, etc.
error: osv-scanner detected a critical severity issue
detail: SEVERITY: CRITICAL in package immer, vulnerable to CVE-2021-23436
error: osv-scanner detected a critical severity issue
detail: SEVERITY: CRITICAL in package json-schema, vulnerable to CVE-2021-3918
error: Plugin version 0.0.9 is invalid.
detail: The submitted plugin version 0.0.9 is not greater than the latest published version 0.0.9 on grafana.com.
```

This can be changed to JSON by passing a configuration file which includes:

```yaml
global:
  jsonOutput: true
```

Resulting in output similar to:

```JSON
{
  "id": "briangann-gauge-panel",
  "version": "0.0.9",
  "plugin-validator": {
    "brokenlinks": [
      {
        "Severity": "warning",
        "Title": "README.md: possible broken link: https://www.d3js.org (404 Not Found)",
        "Detail": "README.md might contain broken links. Check that all links are valid and publicly accessible.",
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
        "Name": "osv-scanner-critical-severity-vulnerabilities-detected"
      },
      {
        "Severity": "error",
        "Title": "osv-scanner detected a critical severity issue",
        "Detail": "SEVERITY: CRITICAL in package json-schema, vulnerable to CVE-2021-3918",
        "Name": "osv-scanner-critical-severity-vulnerabilities-detected"
      },
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

### Severity

By default, the tool will show any warning or error level results from the analyzers. To see all results including successes, you can pass a configuration file which includes:

```yaml
global:
  reportAll: true
```

## Getting Help

- :open_book: Check out our plugin [documentation](https://grafana.com/developers/plugin-tools).
- :handshake: Join the [community forum](https://community.grafana.com/tag/plugins).
- :speech_balloon: Chat to us in the Grafana Slack [#plugins channel](https://grafana.slack.com/archives/C3HJV5PNE).
- :memo: [File an issue](https://github.com/grafana/plugin-validator/issues) for any bugs or feature requests.

## License

Grafana Plugin Validator is distributed under the [Apache 2.0 License](https://github.com/grafana/plugin-validator/blob/master/LICENSE).
