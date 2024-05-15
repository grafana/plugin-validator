package githubapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/grafana/plugin-validator/pkg/logme"
)

func FetchLatestGrafanaSdkRelease() (Release, error) {
	return FetchGrafanaSdkReleaseByTag("latest")
}

func FetchGrafanaSdkReleaseByTag(tag string) (Release, error) {
	if tag == "" {
		return Release{}, fmt.Errorf("tag is required")
	}

	apiUrl := "https://api.github.com/repos/grafana/grafana-plugin-sdk-go/releases"

	if tag == "latest" {
		apiUrl = apiUrl + "/latest"
	} else {
		apiUrl = apiUrl + "/tags/" + tag
	}

	req, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return Release{}, err
	}

	token := os.Getenv("GITHUB_TOKEN")

	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Release{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var release Release

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return Release{}, err
	}

	if release.TagName == "" {

		logme.Debugln(
			fmt.Sprintf(
				"Github returned no releases. This is unexpected. Github status code: %d",
				resp.StatusCode,
			),
		)
		logme.Debugln(resp.Body)

		return Release{}, fmt.Errorf(
			"Github returned no releases. This is unexpected. Github status code: %d",
			resp.StatusCode,
		)
	}

	return release, nil
}
