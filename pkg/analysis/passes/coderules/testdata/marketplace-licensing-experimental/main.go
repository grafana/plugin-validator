package main

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend/app"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
)

func main() {
	datasource.Manage("datasource-id", newDatasource, datasource.ManageOpts{
		Licensing: marketplace.Licensing(marketplace.Opts{}),
	})
	app.Manage("app-id", newApp, app.ManageOpts{
		Licensing: marketplace.Licensing(marketplace.Opts{ParentPluginID: "parent-app-id"}),
	})
}
