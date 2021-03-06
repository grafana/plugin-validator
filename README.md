# Grafana Plugin Validator

[![License](https://img.shields.io/github/license/grafana/plugin-validator)](LICENSE)

A tool for validating community plugins for publishing to Grafana.com.

The tool expects path to either a remote or a local ZIP archive.

## Install

```
cd cmd/plugincheck
go install
```

## Run

Validate a remote archive:

```
plugincheck https://github.com/marcusolsson/grafana-jsonapi-datasource/releases/download/v0.6.0/marcusolsson-json-datasource-0.6.0.zip
```

Validate a local plugin archive:

```
plugincheck ./marcusolsson-json-datasource-0.6.0.zip
```

## License

Grafana Plugin Validator is distributed under the [Apache 2.0 License](https://github.com/grafana/plugin-validator/blob/master/LICENSE).
