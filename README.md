# Grafana Plugin Validator

[![License](https://img.shields.io/github/license/grafana/plugin-validator)](LICENSE)

A tool for validating community plugins for publishing to Grafana.com.

Currently only supports plugins hosted on GitHub.

## Local validation of plugin.json

Adding `https://raw.githubusercontent.com/grafana/plugin-validator/master/config/plugin.schema.json` as `$schema` of the `plugin.json` will help you to locally validate the file without installing the **plugincheck**. This will also allows you to improve the `plugin.json` authoring experience with auto complete in IDEs.

## Install

```
cd cmd/plugincheck
go install
```

## Run

```
plugincheck -url https://github.com/grafana/worldmap-panel
```

## License

Grafana Plugin Validator is distributed under the [Apache 2.0 License](https://github.com/grafana/plugin-validator/blob/master/LICENSE).
