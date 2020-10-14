package plugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/grafana/plugin-validator/pkg/grafana"
	"github.com/xeipuuv/gojsonschema"
)

type largeFileChecker struct{}

func (c largeFileChecker) check(ctx *checkContext) ([]ValidationComment, error) {
	var errs []ValidationComment

	filepath.Walk(ctx.DistDir, func(path string, info os.FileInfo, err error) error {
		if info.Size() > 1000000 {
			b, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			if !strings.HasPrefix(http.DetectContentType(b), "text/plain") {
				errs = append(errs, ValidationComment{
					Severity: "error",
					Message:  fmt.Sprintf("File is too large: %s", strings.TrimPrefix(strings.TrimPrefix(path, ctx.DistDir), "/")),
					Details:  "Due to restrictions in the GitHub API, we're currently not able to publish plugins that contain files that are larger than 1 MB.",
				})
			}

		}
		return nil
	})

	return errs, nil
}

type linkChecker struct{}

func (c linkChecker) check(ctx *checkContext) ([]ValidationComment, error) {
	var errs []ValidationComment

	mdLinks := regexp.MustCompile(`\[.+?\]\((.+?)\)`)

	matches := mdLinks.FindAllSubmatch(ctx.Readme, -1)

	var urls []string
	for _, m := range matches {
		path := string(m[1])

		if strings.HasPrefix(path, "#") {
			// Named anchors are allowed, but not checked.
			continue
		}

		// Strip optional alt text for images, e.g. ![image](./path/to/image "alt text").
		fields := strings.Fields(path)
		if len(fields) > 0 {
			path = fields[0]
		}

		if strings.HasPrefix(path, "https://") || strings.HasPrefix(path, "http://") {
			urls = append(urls, path)
		} else {
			errs = append(errs, ValidationComment{
				Severity: checkSeverityError,
				Message:  "README contains a relative link",
				Details:  fmt.Sprintf("Relative links are not supported by Grafana and results in broken links wherever we display the README. Please convert `%s` into an absolute link.", path),
			})
		}
	}

	type urlstatus struct {
		url    string
		status string
	}

	brokenCh := make(chan urlstatus)

	var wg sync.WaitGroup
	wg.Add(len(urls))

	for _, u := range urls {
		go func(url string) {
			defer wg.Done()

			resp, err := http.Get(url)
			if err != nil {
				brokenCh <- urlstatus{url: url, status: err.Error()}
				return
			}

			if resp.StatusCode != http.StatusOK {
				brokenCh <- urlstatus{url: url, status: resp.Status}
			}
		}(u)
	}

	go func() {
		wg.Wait()
		close(brokenCh)
	}()

	for link := range brokenCh {
		errs = append(errs, ValidationComment{
			Severity: checkSeverityError,
			Message:  "README contains a broken link",
			Details:  fmt.Sprintf("Something went wrong when we tried looking up [%s](%s) (`%s`).", link.url, link.url, link.status),
		})
	}

	return errs, nil
}

type screenshotChecker struct{}

func (c screenshotChecker) check(ctx *checkContext) ([]ValidationComment, error) {

	var data struct {
		Info struct {
			Screenshots []struct {
				Name string `json:"name"`
				Path string `json:"path"`
			} `json:"screenshots"`
		} `json:"info"`
	}
	if err := json.Unmarshal(ctx.Metadata, &data); err != nil {
		return nil, nil
	}

	if len(data.Info.Screenshots) == 0 {
		return []ValidationComment{
			{
				Severity: checkSeverityWarning,
				Message:  "Plugin is missing screenshots",
				Details:  "Screenshots help users understand what your plugin does, and how to use it. Consider providing screenshots to your plugin by adding them under `info.screenshots` in the `plugin.json` file. For more information, refer to the [reference documentation](https://grafana.com/docs/grafana/latest/developers/plugins/metadata/#screenshots).",
			},
		}, nil
	}

	var errs []ValidationComment
	for _, ss := range data.Info.Screenshots {
		comment, ok := checkRelativePath(ctx, ss.Path)
		if !ok {
			errs = append(errs, comment)
		}
	}

	return errs, nil
}

type developerJargonChecker struct{}

// check checks whether the README contains developer jargon.
func (c developerJargonChecker) check(ctx *checkContext) ([]ValidationComment, error) {
	jargon := []string{
		"yarn",
		"nodejs",
	}

	var found []string
	for _, word := range jargon {
		if bytes.Contains(ctx.Readme, []byte(word)) {
			found = append(found, word)
		}
	}

	if len(found) > 0 {
		return []ValidationComment{
			{
				Severity: checkSeverityWarning,
				Message:  "README contains developer jargon",
				Details:  "Grafana uses the README within the application to help users understand how to use your plugin. Instructions for building and testing the plugin can be confusing for the end user. You can maintain separate instructions for users and developers by replacing the README in the dist directory with the user documentation.",
			},
		}, nil
	}

	return nil, nil
}

type distExistsChecker struct{}

func (c *distExistsChecker) check(ctx *checkContext) ([]ValidationComment, error) {
	var errs []ValidationComment

	_, err := os.Stat(ctx.DistDir)
	if err != nil {
		if os.IsNotExist(err) {
			errs = append(errs, ValidationComment{
				Severity: checkSeverityError,
				Message:  "Missing dist directory",
				Details:  "Grafana requires a production build of your plugin. Run `yarn build` and `git add -f dist/` in your release branch to add the production build.",
			})
			return errs, nil
		}
		return nil, err
	}

	return errs, nil
}

type pluginIDHasTypeSuffixChecker struct{}

// check checks that the type in the plugin ID is the same as the type defined
// in plugin.json.
func (c *pluginIDHasTypeSuffixChecker) check(ctx *checkContext) ([]ValidationComment, error) {
	var data struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(ctx.Metadata, &data); err != nil {
		return nil, err
	}

	if data.Type == "" {
		return nil, nil
	}

	idParts := strings.Split(data.ID, "-")

	if idParts[len(idParts)-1] != data.Type {
		return []ValidationComment{
			{
				Severity: checkSeverityError,
				Message:  "Plugin ID and type doesn't match",
				Details:  fmt.Sprintf(`The plugin ID must end with the plugin type. Add "-%s" at the end of your plugin ID.`, data.Type),
			},
		}, nil
	}

	return nil, nil
}

type pluginIDFormatChecker struct{}

// check checks whether the plugin ID follows the naming conventions.
func (c *pluginIDFormatChecker) check(ctx *checkContext) ([]ValidationComment, error) {
	var data struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(ctx.Metadata, &data); err != nil {
		return nil, err
	}

	var errs []ValidationComment

	if data.ID != "" {
		parts := len(strings.Split(data.ID, "-"))
		if parts < 2 || parts > 3 {
			errs = append(errs, ValidationComment{
				Severity: checkSeverityError,
				Message:  "Invalid ID format",
				Details:  "A plugin ID must have the form `<username>-<name>-<type>` or `<username>-<type>`, where\n\n- `username` is the [Grafana.com](https://grafana.com) account that owns the plugin\n- `name` is the name of the plugin\n- `type` is the type of the plugin and must be one of `panel`, `datasource`, or `app`",
			})
		}
	}

	return errs, nil
}

type pluginNameChecker struct{}

// check checks whether the plugin ID and name are the same.
func (c *pluginNameChecker) check(ctx *checkContext) ([]ValidationComment, error) {
	var data struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal(ctx.Metadata, &data); err != nil {
		return nil, err
	}

	var errs []ValidationComment

	if data.ID != "" && data.Name != "" && data.ID == data.Name {
		errs = append(errs, ValidationComment{
			Severity: checkSeverityWarning,
			Message:  "Plugin name and ID are the same",
			Details:  "While the `id` property must be readable by a machine, the `name` of a plugin should be human-friendly.",
		})
	}

	return errs, nil
}

type jsonSchemaChecker struct {
	schema string
}

// check validates the plugin.json file against a JSON Schema.
func (c *jsonSchemaChecker) check(ctx *checkContext) ([]ValidationComment, error) {
	var errs []ValidationComment

	// gojsonschema requires absolute path to the schema.
	schemaPath, err := filepath.Abs(c.schema)
	if err != nil {
		return nil, err
	}

	schemaLoader := gojsonschema.NewReferenceLoader("file://" + schemaPath)
	documentLoader := gojsonschema.NewReferenceLoader("file://" + ctx.MetadataPath)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return nil, err
	}

	if !result.Valid() {
		for _, desc := range result.Errors() {
			errs = append(errs, ValidationComment{
				Severity: checkSeverityError,
				Message:  "Invalid plugin.json",
				Details:  fmt.Sprintf("`%s`: %s\n\nFor more information, refer to the [reference documentation](https://grafana.com/docs/grafana/latest/developers/plugins/metadata/).", desc.Field(), desc.Description()),
			})
		}
	}

	return errs, nil
}

type packageVersionMatchChecker struct {
	schema string
}

// check checks that the version specified in package.json is the same as the
// version in plugin.json.
func (c *packageVersionMatchChecker) check(ctx *checkContext) ([]ValidationComment, error) {
	packageFile, err := ioutil.ReadFile(filepath.Join(ctx.RootDir, "package.json"))
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(packageFile, &pkg); err != nil {
		return nil, err
	}

	pluginFile, err := ioutil.ReadFile(filepath.Join(ctx.DistDir, "plugin.json"))
	if err != nil {
		return nil, err
	}

	var plugin struct {
		Info struct {
			Version string `json:"version"`
		} `json:"info"`
	}
	if err := json.Unmarshal(pluginFile, &plugin); err != nil {
		return nil, err
	}

	if plugin.Info.Version != pkg.Version {
		return []ValidationComment{
			{
				Severity: checkSeverityError,
				Message:  "Mismatched package version",
				Details:  "The `version` in `package.json` needs to match the `info.version` in `plugin.json. Set `info.version` in `plugin.json` to `%VERSION%` to use the version found in package.json when building the plugin.",
			},
		}, nil
	}

	return nil, nil
}

type logosExistChecker struct{}

// check checks whether the specified logos exists.
func (c *logosExistChecker) check(ctx *checkContext) ([]ValidationComment, error) {
	path, err := fallbackDir("plugin.json", ctx.DistDir, ctx.SrcDir)
	if err != nil {
		if err == errFileNotFound {
			return nil, nil
		}
		return nil, err
	}

	pluginFile, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var plugin struct {
		Info struct {
			Logos struct {
				Small string `json:"small"`
				Large string `json:"large"`
			} `json:"logos"`
		} `json:"info"`
	}
	if err := json.Unmarshal(pluginFile, &plugin); err != nil {
		return nil, err
	}

	var errs []ValidationComment

	// Check for small logo.
	if plugin.Info.Logos.Small != "" {
		comment, ok := checkRelativePath(ctx, plugin.Info.Logos.Small)
		if !ok {
			errs = append(errs, comment)
		}
	}

	// Check for large logo.
	if plugin.Info.Logos.Large != "" {
		comment, ok := checkRelativePath(ctx, plugin.Info.Logos.Large)
		if !ok {
			errs = append(errs, comment)
		}
	}

	return errs, nil
}

type orgExistsChecker struct {
	username string
	client   *grafana.Client
}

// check checks whether a Grafana.com account exists for a given username.
func (c *orgExistsChecker) check(ctx *checkContext) ([]ValidationComment, error) {
	_, err := c.client.FindOrgBySlug(c.username)
	if err != nil {
		if err == grafana.ErrOrganizationNotFound {
			return []ValidationComment{
				{
					Severity: checkSeverityError,
					Message:  "Missing Grafana.com account",
					Details:  fmt.Sprintf("The first part of the plugin ID must be a valid Grafana.com organization or user. [Sign up on Grafana.com](https://grafana.com/signup/starter/connect-account) to claim **%s**.", c.username),
				},
			}, nil
		} else if err == grafana.ErrPrivateOrganization {
			return nil, nil
		}
		return nil, err
	}
	return nil, nil
}

type templateReadmeChecker struct{}

// check checks whether a Grafana.com account exists for a given username.
func (c *templateReadmeChecker) check(ctx *checkContext) ([]ValidationComment, error) {
	re := regexp.MustCompile("^# Grafana (Panel|Data Source|Data Source Backend) Plugin Template")
	m := re.Find(ctx.Readme)
	if m != nil {
		return []ValidationComment{
			{
				Severity: checkSeverityWarning,
				Message:  "Found template README.md",
				Details:  "It looks like you haven't updated the README.md that was provided by the plugin template. Update the README with information about your plugin and how to use it.",
			},
		}, nil
	}
	return nil, nil
}

type pluginPlatformChecker struct{}

// check checks whether a Grafana.com account exists for a given username.
func (c *pluginPlatformChecker) check(ctx *checkContext) ([]ValidationComment, error) {
	var modulePath string
	var err error
	modulePath, err = fallbackDir("module.ts", ctx.DistDir, ctx.SrcDir)
	if err != nil {
		modulePath, err = fallbackDir("module.js", ctx.DistDir, ctx.SrcDir)
		if err != nil {
			return nil, nil
		}
	}

	b, err := ioutil.ReadFile(modulePath)
	if err != nil {

	}

	reactExp := regexp.MustCompile(`(DataSourcePlugin|PanelPlugin)`)
	angularExp := regexp.MustCompile(`\s(PanelCtrl|QueryCtrl|QueryOptionsCtrl|ConfigCtrl)`)

	if angularExp.Match(b) && !reactExp.Match(b) {
		return []ValidationComment{
			{
				Severity: checkSeverityWarning,
				Message:  "Plugin uses legacy platform",
				Details:  "Grafana 7.0 introduced a new plugin platform based on [ReactJS](https://reactjs.org/). We currently have no plans of removing support for Angular-based plugins, but we encourage you migrate your plugin to the new platform.",
			},
		}, nil
	}

	return nil, nil
}

var errFileNotFound = errors.New("file not found")

// fallbackDir looks for a filename in a number of directories, and returns the
// path to the first path to exist.
func fallbackDir(filename string, dirs ...string) (string, error) {
	for _, dir := range dirs {
		path := filepath.Join(dir, filename)
		ok, err := fileExists(path)
		if err != nil {
			return "", err
		}
		if ok {
			return path, nil
		}
	}
	return "", errFileNotFound
}

func fileExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func checkRelativePath(ctx *checkContext, path string) (ValidationComment, bool) {
	u, err := url.Parse(path)
	if err != nil {
		return ValidationComment{
			Severity: checkSeverityError,
			Message:  fmt.Sprintf("Invalid path: %s", path),
			Details:  "Paths need to be relative to the plugin.json file, and can't begin with `.` or `/`. For example, `img/screenshot.png`.",
		}, false
	}

	if strings.HasPrefix(path, ".") || strings.HasPrefix(path, "/") || u.IsAbs() {
		return ValidationComment{
			Severity: checkSeverityError,
			Message:  fmt.Sprintf("Invalid path: %s", path),
			Details:  "Paths need to be relative to the plugin.json file, and can't begin with `.` or `/`. For example, `img/screenshot.png`.",
		}, false
	}

	_, err = fallbackDir(path, ctx.DistDir, ctx.SrcDir)
	if err != nil {
		if err == errFileNotFound {
			return ValidationComment{
				Severity: checkSeverityError,
				Message:  fmt.Sprintf("File not found: %s", path),
				Details:  "We couldn't find the specified file. Make sure that the file exists.",
			}, false
		}
	}

	return ValidationComment{}, true
}
