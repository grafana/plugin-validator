package main

import (
	"github.com/grafana/grafana-plugin-sdk-go/backend/app"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	experimentalmarketplace "github.com/grafana/grafana-plugin-sdk-go/experimental/marketplace"
	"github.com/grafana/grafana-plugin-sdk-go/marketplace"
)

func main() {
	datasource.Manage("datasource-id", newDatasource, datasource.ManageOpts{
		Licensing: experimentalmarketplace.Licensing(experimentalmarketplace.Opts{}),
	})
	app.Manage("app-id", newApp, app.ManageOpts{
		Licensing: marketplace.Licensing(marketplace.Opts{}),
	})
	app.Manage("other-app-id", newApp, app.ManageOpts{})
}
