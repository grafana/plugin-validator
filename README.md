# Grafana Plugin Validator

[![License](https://img.shields.io/github/license/grafana/plugin-validator)](LICENSE)

A tool for validating community plugins for publishing to Grafana.com.

Currently only supports plugins hosted on GitHub.

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
