# Grafana Plugin Validator

[![License](https://img.shields.io/github/license/grafana/plugin-validator)](LICENSE)

A tool for validating community plugins for publishing to Grafana.com.

The tool expects path to either a remote or a local ZIP archive.

## Install

```
cd cmd/plugincheck
go install
```

Alternative: using docker

```
docker pull ghcr.io/grafana/plugin-validator:v0.6.2
```

## Run

Validate a remote archive:

```
plugincheck https://github.com/marcusolsson/grafana-jsonapi-datasource/releases/download/v0.6.0/marcusolsson-json-datasource-0.6.0.zip
```

Validate a remote archive using docker:

```
docker run --rm ghcr.io/grafana/plugin-validator:v0.6.2 https://github.com/marcusolsson/grafana-jsonapi-datasource/releases/download/v0.6.0/marcusolsson-json-datasource-0.6.0.zip
```

Validate a local plugin archive:

```
plugincheck ./marcusolsson-json-datasource-0.6.0.zip
```

Validate a local plugin archive using docker:

```
docker run --rm -v $(pwd):/plugin ghcr.io/grafana/plugin-validator:v0.6.2 /plugin/marcusolsson-json-datasource-0.6.0.zip
```

## License

Grafana Plugin Validator is distributed under the [Apache 2.0 License](https://github.com/grafana/plugin-validator/blob/master/LICENSE).
