package main

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend/app"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
)

func main() {
	datasource.Manage("datasource-id", newDatasource, datasource.ManageOpts{})
	app.Manage("app-id", newApp, app.ManageOpts{})
	datasource.Manage("datasource-id", newDatasource, datasource.ManageOpts{Licensing: nil})
}
