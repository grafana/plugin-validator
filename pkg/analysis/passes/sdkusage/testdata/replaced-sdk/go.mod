module github.com/grafana/grafana-testing-replaced-sdk-datasource

go 1.22

require (
	github.com/grafana/grafana-plugin-sdk-go v0.260.3
)

replace github.com/grafana/grafana-plugin-sdk-go v0.260.3 => ./grafana-plugin-sdk-go
