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
You must pass a configuration file to the validator with the `-config` option. Several configuraton examples are available to use here https://github.com/grafana/plugin-validator/tree/main/config


### Enabling and disabling checks

If you wish to disable an specific check (analyzer) you can define this in your configuration file adding an `analyzers` section and specyfing which analyzer or analyzer rules to enable and disable.

E.g.: disable the `version` analyzer

```yaml
global:
  enabled: true
  jsonOutput: false
  reportAll: false

analyzers:
  version:
    enabled: true
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

## License

Grafana Plugin Validator is distributed under the [Apache 2.0 License](https://github.com/grafana/plugin-validator/blob/master/LICENSE).
