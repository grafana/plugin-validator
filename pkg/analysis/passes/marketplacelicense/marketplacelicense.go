package marketplacelicense

import (
	"bufio"
	"bytes"
	"debug/buildinfo"
	"fmt"
	"go/ast"
	"go/build"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/mod/modfile"

	"github.com/grafana/plugin-validator/pkg/analysis"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/archive"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/nestedmetadata"
	"github.com/grafana/plugin-validator/pkg/analysis/passes/sourcecode"
)

const (
	marketplaceDevelopmentTag = "marketplace_dev"
	marketplaceSDKModule      = "github.com/grafana/grafana-plugin-sdk-go"
	marketplacePackage        = marketplaceSDKModule + "/experimental/marketplace"
	maxSourceFileSize         = 8 * 1024 * 1024
	maxGoModSize              = 2 * 1024 * 1024
)

var (
	marketplaceDevelopmentBuild = &analysis.Rule{
		Name:     "marketplace-development-build",
		Severity: analysis.Error,
	}
	marketplaceBuildUnverified = &analysis.Rule{
		Name:     "marketplace-build-unverified",
		Severity: analysis.Error,
	}
)

var Analyzer = &analysis.Analyzer{
	Name: "marketplacelicense",
	Requires: []*analysis.Analyzer{
		archive.Analyzer,
		nestedmetadata.Analyzer,
		sourcecode.Analyzer,
	},
	Run: run,
	Rules: []*analysis.Rule{
		marketplaceDevelopmentBuild,
		marketplaceBuildUnverified,
	},
	ReadmeInfo: analysis.ReadmeInfo{
		Name:        "Marketplace License",
		Description: "Rejects backend executables built with marketplace development-mode license checks disabled.",
	},
}

type targetBuild struct {
	goos   string
	goarch string
	tags   map[string]bool
}

type binaryInspection struct {
	buildInfoReadable bool
	developmentTag    bool
	sdkDependency     bool
	sdkVersion        string
	sdkReplaced       bool
	buildInfoError    error
	target            targetBuild
}

func run(pass *analysis.Pass) (interface{}, error) {
	archiveDir, ok := pass.ResultOf[archive.Analyzer].(string)
	if !ok {
		reportUnverified(
			pass,
			"Could not discover marketplace backend executables",
			"The plugin archive structure could not be determined",
		)
		return nil, nil
	}

	metadataMap, ok := pass.ResultOf[nestedmetadata.Analyzer].(nestedmetadata.Metadatamap)
	if !ok {
		reportUnverified(
			pass,
			"Could not discover marketplace backend executables",
			"The plugin metadata could not be read",
		)
		return nil, nil
	}

	sourceCodeDir := marketplaceSourceDirectory(pass)

	pluginJSONPaths := make([]string, 0, len(metadataMap))
	for pluginJSONPath := range metadataMap {
		pluginJSONPaths = append(pluginJSONPaths, pluginJSONPath)
	}
	sort.Strings(pluginJSONPaths)

	for _, pluginJSONPath := range pluginJSONPaths {
		data := metadataMap[pluginJSONPath]
		if !data.Backend {
			if data.Executable != "" {
				reportUnverified(
					pass,
					fmt.Sprintf("Could not verify backend declaration for %s", pluginJSONPath),
					"plugin.json declares an executable but sets backend to false",
				)
			}
			continue
		}
		if data.Executable == "" {
			reportUnverified(
				pass,
				fmt.Sprintf("Could not discover backend executables for %s", pluginJSONPath),
				"The backend plugin does not declare an executable",
			)
			continue
		}

		executableParentDir := filepath.Join(
			archiveDir,
			filepath.Dir(pluginJSONPath),
			filepath.Dir(data.Executable),
		)
		if !pathWithinRoot(archiveDir, executableParentDir) {
			reportUnverified(
				pass,
				fmt.Sprintf("Could not verify marketplace license mode for %s", pluginJSONPath),
				"The backend executable path resolves outside the plugin archive",
			)
			continue
		}

		safeExecutableParentDir, err := resolveArchiveDirectory(archiveDir, executableParentDir)
		if err != nil {
			reportUnverified(
				pass,
				fmt.Sprintf("Could not discover backend executables for %s", pluginJSONPath),
				err.Error(),
			)
			continue
		}

		entries, err := os.ReadDir(safeExecutableParentDir)
		if err != nil {
			reportUnverified(
				pass,
				fmt.Sprintf("Could not discover backend executables for %s", pluginJSONPath),
				"The backend executable directory could not be read: "+err.Error(),
			)
			continue
		}

		executableBase := filepath.Base(data.Executable)
		binaries := make([]string, 0)
		for _, entry := range entries {
			if entry.IsDir() || !isExecutableCandidate(executableBase, entry.Name()) {
				continue
			}
			binaries = append(binaries, filepath.Join(safeExecutableParentDir, entry.Name()))
		}
		sort.Strings(binaries)
		if len(binaries) == 0 {
			reportUnverified(
				pass,
				fmt.Sprintf("Could not discover backend executables for %s", pluginJSONPath),
				"No packaged backend executable matched the executable declared in plugin.json",
			)
			continue
		}

		for _, binaryPath := range binaries {
			inspectAndReport(pass, archiveDir, binaryPath, sourceCodeDir)
		}
	}

	return nil, nil
}

func reportUnverified(pass *analysis.Pass, title, detail string) {
	pass.ReportResult(
		pass.AnalyzerName,
		marketplaceBuildUnverified,
		title,
		detail+". Manual review is required before submission can proceed.",
	)
}

func marketplaceSourceDirectory(pass *analysis.Pass) string {
	sourceCodeDir, _ := pass.ResultOf[sourcecode.Analyzer].(string)
	return sourceCodeDir
}

func inspectSourceSDK(pass *analysis.Pass, sourceCodeDir string) string {
	goModPath := filepath.Join(sourceCodeDir, "go.mod")
	contents, err := readConfinedRegularFile(sourceCodeDir, goModPath, maxGoModSize)
	if err != nil {
		reportUnverified(
			pass,
			"Could not verify the marketplace SDK dependency in source code",
			"go.mod could not be read: "+err.Error(),
		)
		return ""
	}
	parsed, err := modfile.Parse(goModPath, contents, nil)
	if err != nil {
		reportUnverified(
			pass,
			"Could not verify the marketplace SDK dependency in source code",
			"go.mod could not be parsed: "+err.Error(),
		)
		return ""
	}

	sdkVersion := ""
	for _, requirement := range parsed.Require {
		if requirement.Mod.Path == marketplaceSDKModule {
			sdkVersion = requirement.Mod.Version
			break
		}
	}
	if sdkVersion == "" {
		reportUnverified(
			pass,
			"Could not verify the marketplace SDK dependency in source code",
			"go.mod does not require the Grafana Plugin SDK",
		)
		return ""
	}

	vendoredSDKPath := filepath.Join(
		sourceCodeDir,
		"vendor",
		filepath.FromSlash(marketplaceSDKModule),
	)
	if _, err := os.Lstat(vendoredSDKPath); err == nil {
		reportUnverified(
			pass,
			"Could not verify the vendored marketplace SDK dependency",
			"Vendored Grafana Plugin SDK source requires an explicit integrity review",
		)
	} else if !os.IsNotExist(err) {
		reportUnverified(
			pass,
			"Could not verify the vendored marketplace SDK dependency",
			"The vendored SDK path could not be inspected: "+err.Error(),
		)
	}

	var selectedReplacement *modfile.Replace
	for _, replacement := range parsed.Replace {
		if replacement.Old.Path != marketplaceSDKModule {
			continue
		}
		if replacement.Old.Version == sdkVersion {
			selectedReplacement = replacement
			break
		}
		if replacement.Old.Version == "" {
			selectedReplacement = replacement
		}
	}
	if selectedReplacement == nil {
		return sdkVersion
	}
	if selectedReplacement.New.Path != marketplaceSDKModule || selectedReplacement.New.Version == "" {
		reportUnverified(
			pass,
			"Unsupported marketplace SDK replacement found in source code",
			"go.mod replaces the Grafana Plugin SDK with a local path or fork",
		)
		return sdkVersion
	}
	return selectedReplacement.New.Version
}

func readConfinedRegularFile(root, path string, maxSize int64) ([]byte, error) {
	if !pathWithinRoot(root, path) {
		return nil, fmt.Errorf("source path resolves outside the trusted checkout")
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve source checkout path: %w", err)
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, fmt.Errorf("resolve source file path: %w", err)
	}
	if !pathWithinRoot(resolvedRoot, resolvedPath) {
		return nil, fmt.Errorf("source file symlink resolves outside the trusted checkout")
	}
	pathInfo, err := os.Stat(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("read source file information: %w", err)
	}
	if !pathInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("source file is not a regular file")
	}
	if pathInfo.Size() > maxSize {
		return nil, fmt.Errorf("source file exceeds the %d-byte inspection limit", maxSize)
	}

	file, err := os.Open(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("open source file: %w", err)
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("read source file information: %w", err)
	}
	if !fileInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("source file is not a regular file")
	}
	if fileInfo.Size() > maxSize {
		return nil, fmt.Errorf("source file exceeds the %d-byte inspection limit", maxSize)
	}
	contents, err := io.ReadAll(io.LimitReader(file, maxSize+1))
	if err != nil {
		return nil, fmt.Errorf("read source file: %w", err)
	}
	if int64(len(contents)) > maxSize {
		return nil, fmt.Errorf("source file exceeds the %d-byte inspection limit", maxSize)
	}
	return contents, nil
}

func sourceUsesMarketplaceLicense(root string, target targetBuild) (bool, bool, error) {
	type sourceFunction struct {
		body        *ast.BlockStmt
		aliases     map[string]struct{}
		dotImported bool
	}
	type sourcePackage struct {
		functions map[string][]sourceFunction
		mains     []sourceFunction
		inits     []sourceFunction
	}

	packages := make(map[string]*sourcePackage)
	referencesMarketplace := false
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "dist", "node_modules", "vendor":
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		contents, err := readConfinedRegularFile(root, path, maxSourceFileSize)
		if err != nil {
			return err
		}
		if !productionBuildEligible(path, contents, target) {
			return nil
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), path, contents, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		if parsed.Name.Name != "main" || (!target.tags["cgo"] && importsC(parsed)) {
			return nil
		}

		aliases := make(map[string]struct{})
		dotImported := false
		for _, importSpec := range parsed.Imports {
			importPath, err := strconv.Unquote(importSpec.Path.Value)
			if err != nil || importPath != marketplacePackage {
				continue
			}
			referencesMarketplace = true
			if importSpec.Name == nil {
				aliases["marketplace"] = struct{}{}
				continue
			}
			switch importSpec.Name.Name {
			case ".":
				dotImported = true
			case "_":
			default:
				aliases[importSpec.Name.Name] = struct{}{}
			}
		}

		packageDir := filepath.Dir(path)
		packageInfo := packages[packageDir]
		if packageInfo == nil {
			packageInfo = &sourcePackage{functions: make(map[string][]sourceFunction)}
			packages[packageDir] = packageInfo
		}
		for _, declaration := range parsed.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil || function.Body == nil {
				continue
			}
			candidate := sourceFunction{
				body:        function.Body,
				aliases:     aliases,
				dotImported: dotImported,
			}
			packageInfo.functions[function.Name.Name] = append(
				packageInfo.functions[function.Name.Name],
				candidate,
			)
			switch function.Name.Name {
			case "main":
				packageInfo.mains = append(packageInfo.mains, candidate)
			case "init":
				packageInfo.inits = append(packageInfo.inits, candidate)
			}
		}
		return nil
	})
	if err != nil {
		return false, false, err
	}

	reachesMarketplaceLicense := func(root sourceFunction, functions map[string][]sourceFunction) bool {
		queue := []sourceFunction{root}
		visited := make(map[*ast.BlockStmt]bool)
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			if visited[current.body] {
				continue
			}
			visited[current.body] = true

			found := false
			calledFunctions := make(map[string]struct{})
			ast.Inspect(current.body, func(node ast.Node) bool {
				if found {
					return false
				}
				if _, ok := node.(*ast.FuncLit); ok {
					return false
				}
				call, ok := node.(*ast.CallExpr)
				if !ok {
					return true
				}
				switch function := call.Fun.(type) {
				case *ast.SelectorExpr:
					identifier, ok := function.X.(*ast.Ident)
					if ok && (identifier.Obj == nil || identifier.Obj.Kind == ast.Pkg) &&
						isMarketplaceLicenseFunction(function.Sel.Name) {
						_, found = current.aliases[identifier.Name]
					}
				case *ast.Ident:
					if function.Obj == nil && current.dotImported &&
						isMarketplaceLicenseFunction(function.Name) {
						found = true
					} else if function.Obj == nil || function.Obj.Kind == ast.Fun {
						calledFunctions[function.Name] = struct{}{}
					}
				}
				return !found
			})
			if found {
				return true
			}
			for functionName := range calledFunctions {
				queue = append(queue, functions[functionName]...)
			}
		}
		return false
	}

	foundMain := false
	for _, packageInfo := range packages {
		if len(packageInfo.mains) == 0 {
			continue
		}
		foundMain = true
		initEnforces := false
		for _, initializer := range packageInfo.inits {
			if reachesMarketplaceLicense(initializer, packageInfo.functions) {
				initEnforces = true
				break
			}
		}
		if initEnforces {
			continue
		}
		for _, mainFunction := range packageInfo.mains {
			if !reachesMarketplaceLicense(mainFunction, packageInfo.functions) {
				return referencesMarketplace, false, nil
			}
		}
	}
	return referencesMarketplace, foundMain, nil
}

func productionBuildEligible(path string, contents []byte, target targetBuild) bool {
	if !filenameMatchesTarget(path, target) {
		return false
	}

	scanner := bufio.NewScanner(bytes.NewReader(contents))
	expressions := make([]constraint.Expr, 0, 1)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "package ") {
			break
		}
		if !strings.HasPrefix(line, "//go:build ") && !strings.HasPrefix(line, "// +build ") {
			continue
		}
		expression, err := constraint.Parse(line)
		if err != nil {
			return false
		}
		expressions = append(expressions, expression)
	}
	if scanner.Err() != nil {
		return false
	}
	for _, expression := range expressions {
		if !expression.Eval(func(tag string) bool {
			return targetHasBuildTag(target, tag)
		}) {
			return false
		}
	}
	return true
}

func filenameMatchesTarget(path string, target targetBuild) bool {
	name := strings.TrimSuffix(filepath.Base(path), ".go")
	if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") {
		return false
	}
	parts := strings.Split(name, "_")
	if len(parts) < 2 {
		return true
	}
	last := parts[len(parts)-1]
	if supportedGOARCH(last) {
		if last != target.goarch {
			return false
		}
		if len(parts) >= 3 {
			penultimate := parts[len(parts)-2]
			if supportedGOOS(penultimate) {
				return penultimate == target.goos
			}
		}
		return true
	}
	if supportedGOOS(last) {
		return last == target.goos
	}
	return true
}

func targetHasBuildTag(target targetBuild, tag string) bool {
	if tag == marketplaceDevelopmentTag || tag == "ignore" {
		return false
	}
	if tag == target.goos || tag == target.goarch || tag == "gc" {
		return true
	}
	if tag == "unix" {
		switch target.goos {
		case "aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos",
			"ios", "linux", "netbsd", "openbsd", "solaris":
			return true
		}
	}
	if target.tags[tag] {
		return true
	}
	for _, releaseTag := range build.Default.ReleaseTags {
		if tag == releaseTag {
			return true
		}
	}
	return false
}

func importsC(file *ast.File) bool {
	for _, importSpec := range file.Imports {
		if importSpec.Path.Value == `"C"` {
			return true
		}
	}
	return false
}

func isMarketplaceLicenseFunction(name string) bool {
	return name == "Manage" || name == "CheckMarketplacePluginLicense"
}

func isExecutableCandidate(base, name string) bool {
	if name == base {
		return true
	}
	if !strings.HasPrefix(name, base+"_") {
		return false
	}

	suffix := strings.TrimPrefix(name, base+"_")
	if strings.HasSuffix(strings.ToLower(suffix), ".exe") {
		suffix = suffix[:len(suffix)-len(".exe")]
	}
	platform := strings.Split(suffix, "_")
	return len(platform) == 2 && supportedGOOS(platform[0]) && supportedGOARCH(platform[1])
}

func supportedGOOS(value string) bool {
	switch value {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "illumos", "ios", "js",
		"linux", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows":
		return true
	default:
		return false
	}
}

func supportedGOARCH(value string) bool {
	switch value {
	case "386", "amd64", "arm", "arm64", "loong64", "mips", "mipsle", "mips64",
		"mips64le", "ppc64", "ppc64le", "riscv64", "s390x", "wasm":
		return true
	default:
		return false
	}
}

func inspectAndReport(
	pass *analysis.Pass,
	archiveDir string,
	binaryPath string,
	sourceCodeDir string,
) {
	displayRoot := archiveDir
	if resolvedRoot, err := filepath.EvalSymlinks(archiveDir); err == nil {
		displayRoot = resolvedRoot
	}
	relativePath, err := filepath.Rel(displayRoot, binaryPath)
	if err != nil {
		relativePath = binaryPath
	}
	relativePath = filepath.ToSlash(relativePath)

	safeBinaryPath, err := resolveArchivePath(archiveDir, binaryPath)
	if err != nil {
		reportUnverified(
			pass,
			fmt.Sprintf("Could not verify marketplace license mode in %s", relativePath),
			err.Error(),
		)
		return
	}

	inspection := inspectBinary(safeBinaryPath)

	if inspection.developmentTag {
		pass.ReportResult(
			pass.AnalyzerName,
			marketplaceDevelopmentBuild,
			fmt.Sprintf("Marketplace development build found in %s", relativePath),
			fmt.Sprintf(
				"The backend executable was built with the %q build tag. Rebuild it without marketplace development mode before submission.",
				marketplaceDevelopmentTag,
			),
		)
		return
	}

	sourceSDKVersion := ""
	if inspection.buildInfoReadable && sourceCodeDir != "" &&
		inspection.target.goos != "" && inspection.target.goarch != "" {
		referencesMarketplace, usesMarketplaceLicense, sourceErr := sourceUsesMarketplaceLicense(
			sourceCodeDir,
			inspection.target,
		)
		if sourceErr != nil {
			reportUnverified(
				pass,
				fmt.Sprintf("Could not inspect marketplace source for %s", relativePath),
				"The source code could not be inspected: "+sourceErr.Error(),
			)
		} else if referencesMarketplace {
			sourceSDKVersion = inspectSourceSDK(pass, sourceCodeDir)
			if !usesMarketplaceLicense {
				reportUnverified(
					pass,
					fmt.Sprintf("Could not verify marketplace license entrypoint for %s", relativePath),
					fmt.Sprintf(
						"No reachable production call to marketplace.Manage or marketplace.CheckMarketplacePluginLicense was found for %s/%s",
						inspection.target.goos,
						inspection.target.goarch,
					),
				)
			}
		}
	}

	sdkVersionMatches := sourceSDKVersion == "" || inspection.sdkVersion == sourceSDKVersion
	if inspection.buildInfoReadable && inspection.sdkDependency &&
		!inspection.sdkReplaced && sdkVersionMatches {
		return
	}

	reasons := make([]string, 0, 4)
	if inspection.buildInfoError != nil {
		reasons = append(reasons, "Go build metadata could not be read: "+inspection.buildInfoError.Error())
	}
	if inspection.buildInfoReadable && !inspection.sdkDependency {
		reasons = append(reasons, "the Grafana Plugin SDK dependency is not present in Go build metadata")
	}
	if inspection.sdkReplaced {
		reasons = append(reasons, "Go build metadata records a local or forked Grafana Plugin SDK replacement")
	}
	if inspection.sdkDependency && !sdkVersionMatches {
		reasons = append(reasons, fmt.Sprintf(
			"the source requires Grafana Plugin SDK %s but the executable records %s",
			sourceSDKVersion,
			inspection.sdkVersion,
		))
	}

	reportUnverified(
		pass,
		fmt.Sprintf("Could not verify marketplace license mode in %s", relativePath),
		strings.Join(reasons, "; "),
	)
}

func pathWithinRoot(root, path string) bool {
	rootPath, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	candidatePath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	relativePath, err := filepath.Rel(rootPath, candidatePath)
	if err != nil {
		return false
	}
	return relativePath != ".." &&
		!strings.HasPrefix(relativePath, ".."+string(filepath.Separator)) &&
		!filepath.IsAbs(relativePath)
}

func resolveArchiveDirectory(root, path string) (string, error) {
	resolvedPath, err := resolveArchiveEntry(root, path, "backend executable directory")
	if err != nil {
		return "", err
	}
	fileInfo, err := os.Stat(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("read backend executable directory information: %w", err)
	}
	if !fileInfo.IsDir() {
		return "", fmt.Errorf("backend executable parent is not a directory")
	}
	return resolvedPath, nil
}

func resolveArchivePath(root, path string) (string, error) {
	resolvedPath, err := resolveArchiveEntry(root, path, "backend executable")
	if err != nil {
		return "", err
	}
	fileInfo, err := os.Stat(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("read backend executable file information: %w", err)
	}
	if !fileInfo.Mode().IsRegular() {
		return "", fmt.Errorf("backend executable is not a regular file")
	}
	return resolvedPath, nil
}

func resolveArchiveEntry(root, path, description string) (string, error) {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve plugin archive path: %w", err)
	}
	if !pathWithinRoot(root, path) && !pathWithinRoot(resolvedRoot, path) {
		return "", fmt.Errorf("%s path resolves outside the plugin archive", description)
	}

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("resolve %s path: %w", description, err)
	}
	if !pathWithinRoot(resolvedRoot, resolvedPath) {
		return "", fmt.Errorf("%s symlink resolves outside the plugin archive", description)
	}
	return resolvedPath, nil
}

func inspectBinary(path string) binaryInspection {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return binaryInspection{buildInfoError: err}
	}

	inspection := binaryInspection{
		buildInfoReadable: true,
		developmentTag:    hasBuildTag(info, marketplaceDevelopmentTag),
		target:            inspectTargetBuild(info),
	}
	inspection.sdkDependency, inspection.sdkVersion, inspection.sdkReplaced = inspectSDKDependency(info)
	return inspection
}

func inspectSDKDependency(info *buildinfo.BuildInfo) (found bool, version string, replaced bool) {
	for _, dependency := range info.Deps {
		if dependency.Path != marketplaceSDKModule {
			continue
		}
		if dependency.Replace == nil {
			return true, dependency.Version, false
		}
		if dependency.Replace.Path != marketplaceSDKModule || dependency.Replace.Version == "" {
			return true, dependency.Version, true
		}
		return true, dependency.Replace.Version, false
	}
	return false, "", false
}

func inspectTargetBuild(info *buildinfo.BuildInfo) targetBuild {
	target := targetBuild{tags: make(map[string]bool)}
	settings := make(map[string]string)
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
		switch setting.Key {
		case "GOOS":
			target.goos = setting.Value
		case "GOARCH":
			target.goarch = setting.Value
		case "CGO_ENABLED":
			target.tags["cgo"] = setting.Value == "1"
		case "GOEXPERIMENT":
			for _, experiment := range splitBuildTags(setting.Value) {
				target.tags["goexperiment."+experiment] = true
			}
		case "-tags":
			for _, tag := range splitBuildTags(setting.Value) {
				target.tags[tag] = true
			}
		}
	}
	addArchitectureBuildTags(&target, settings)
	return target
}

func addArchitectureBuildTags(target *targetBuild, settings map[string]string) {
	featureSettings := map[string]string{
		"386":      "GO386",
		"amd64":    "GOAMD64",
		"arm":      "GOARM",
		"arm64":    "GOARM64",
		"mips":     "GOMIPS",
		"mips64":   "GOMIPS64",
		"mips64le": "GOMIPS64",
		"mipsle":   "GOMIPS",
		"ppc64":    "GOPPC64",
		"ppc64le":  "GOPPC64",
		"riscv64":  "GORISCV64",
	}
	settingKey := featureSettings[target.goarch]
	value := strings.Split(settings[settingKey], ",")[0]
	if settingKey == "" || value == "" {
		return
	}

	target.tags[target.goarch+"."+value] = true
	if target.goarch != "amd64" || len(value) != 2 || value[0] != 'v' {
		return
	}
	level := int(value[1] - '0')
	for current := 1; current <= level && current <= 4; current++ {
		target.tags[fmt.Sprintf("amd64.v%d", current)] = true
	}
}

func hasBuildTag(info *buildinfo.BuildInfo, wanted string) bool {
	for _, setting := range info.Settings {
		if setting.Key != "-tags" {
			continue
		}
		for _, tag := range splitBuildTags(setting.Value) {
			if tag == wanted {
				return true
			}
		}
	}
	return false
}

func splitBuildTags(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || unicode.IsSpace(r)
	})
}
