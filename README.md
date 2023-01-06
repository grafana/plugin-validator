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

## License

Grafana Plugin Validator is distributed under the [Apache 2.0 License](https://github.com/grafana/plugin-validator/blob/master/LICENSE).
