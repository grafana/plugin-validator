package archivetool

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

// testPluginZip builds a minimal plugin archive in memory so the tests
// don't depend on downloading a real release over the network.
func testPluginZip(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("myorg-plugin/plugin.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte(`{"id": "myorg-plugin"}`)); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestArchive(t *testing.T) {
	zipContent := testPluginZip(t)
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(zipContent)
		}),
	)
	defer server.Close()
	pluginToCheck := server.URL + "/myorg-plugin-1.0.0.zip"

	Convey("Test Archive", t, func() {
		Convey("Test readArchive", func() {
			content, err := ReadArchive(pluginToCheck)
			So(err, ShouldBeNil)
			So(len(content), ShouldBeGreaterThan, 0)
		})
		Convey("Test extractPlugin", func() {
			content, err := ReadArchive(pluginToCheck)
			So(err, ShouldBeNil)
			So(len(content), ShouldBeGreaterThan, 0)
			archiveDir, cleanup, err := ExtractPlugin(bytes.NewReader(content))
			So(err, ShouldBeNil)
			So(archiveDir, ShouldStartWith, "/")
			So(cleanup, ShouldNotBeNil)
		})
	})
}
