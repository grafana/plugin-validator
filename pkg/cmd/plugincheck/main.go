package main

import (
	"fmt"
	"os"
)

func main() {

	fmt.Println(`
     888                                               888                888
     888                                               888                888
     888                                               888                888
 .d88888 .d88b. 88888b. 888d888 .d88b.  .d8888b 8888b. 888888 .d88b.  .d88888
d88" 888d8P  Y8b888 "88b888P"  d8P  Y8bd88P"       "88b888   d8P  Y8bd88" 888
888  88888888888888  888888    88888888888     .d888888888   88888888888  888
Y88b 888Y8b.    888 d88P888    Y8b.    Y88b.   888  888Y88b. Y8b.    Y88b 888
 "Y88888 "Y8888 88888P" 888     "Y8888  "Y8888P"Y888888 "Y888 "Y8888  "Y88888
                888
                888
                888

	**plugincheck V1 is no longer supported.**

	Use V2 instead: plugincheck2

	To install it see README https://github.com/grafana/plugin-validator`)
	fmt.Println()

	if isGithubCi() {

		fmt.Println(`

		You are running plugincheck in a Github Action.

		Replace your github action plugincheck-related code with the following:

		- name: Lint plugin
        run: |
          git clone https://github.com/grafana/plugin-validator
          pushd ./plugin-validator/pkg/cmd/plugincheck2
          go install
          popd
          plugincheck2 ${{ steps.metadata.outputs.archive }}

		`)
	}
	os.Exit(1)

}

func isGithubCi() bool {
	return os.Getenv("GITHUB_ACTIONS") == "true"
}
