package main

import (
	"bytes"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestArchive(t *testing.T) {
	Convey("Test Archive", t, func() {
		pluginToCheck := "https://github.com/marcusolsson/grafana-jsonapi-datasource/releases/download/v0.6.0/marcusolsson-json-datasource-0.6.0.zip"
		Convey("Test readArchive", func() {
			content, err := readArchive(pluginToCheck)
			So(err, ShouldBeNil)
			So(len(content), ShouldBeGreaterThan, 0)
		})
		Convey("Test extractPlugin", func() {
			content, err := readArchive(pluginToCheck)
			So(err, ShouldBeNil)
			So(len(content), ShouldBeGreaterThan, 0)
			bytes.NewReader(content)
			archiveDir, cleanup, err := extractPlugin(bytes.NewReader(content))
			So(err, ShouldBeNil)
			So(archiveDir, ShouldStartWith, "/")
			So(cleanup, ShouldNotBeNil)
		})
	})
}
