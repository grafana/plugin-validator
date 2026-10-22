package main

import (
	sdkapp "github.com/grafana/grafana-plugin-sdk-go/backend/app"
	sdkds "github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
	mp "github.com/grafana/grafana-plugin-sdk-go/marketplace"
)

func main() {
	sdkds.Manage("datasource-id", newDatasource, sdkds.ManageOpts{
		Licensing: mp.Licensing(mp.Opts{}),
	})
	sdkapp.Manage("app-id", newApp, sdkapp.ManageOpts{
		Licensing: mp.Licensing(mp.Opts{}),
	})
}
