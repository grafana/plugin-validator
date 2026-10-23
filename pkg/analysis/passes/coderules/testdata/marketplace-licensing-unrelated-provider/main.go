package main

import (
	"example.com/marketplace"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
)

func main() {
	datasource.Manage("datasource-id", newDatasource, datasource.ManageOpts{
		Licensing: marketplace.Licensing(marketplace.Opts{}),
	})
}
