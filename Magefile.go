//go:build mage
// +build mage

package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

type Build mg.Namespace
type Test mg.Namespace
type Docker mg.Namespace
type Run mg.Namespace
type Gen mg.Namespace

const imageName = "grafana/plugin-validator-cli"
const imageVersion = "v2"

var archTargets = map[string]map[string]string{
	"darwin_amd64": {
		"CGO_ENABLED": "0",
		"GO111MODULE": "on",
		"GOARCH":      "amd64",
		"GOOS":        "darwin",
	},
	"darwin_arm64": {
		"CGO_ENABLED": "0",
		"GO111MODULE": "on",
		"GOARCH":      "arm64",
		"GOOS":        "darwin",
	},
	"linux_amd64": {
		"CGO_ENABLED": "0",
		"GO111MODULE": "on",
		"GOARCH":      "amd64",
		"GOOS":        "linux",
	},
}

// Default target to run when none is specified
// If not set, running mage will list available targets
var Default = Build.Local

/* Docker */
func buildDockerImage() error {
	return sh.RunV(
		"docker",
		"build",
		"--no-cache",
		"--pull",
		"-t",
		imageName+":"+imageVersion,
		"-t",
		imageName+":latest",
		"-f",
		"Dockerfile",
		".",
	)
}

func pushDockerImage() error {
	if err := sh.RunV("docker", "push", imageName+":"+imageVersion); err != nil {
		return err
	}
	if err := sh.RunV("docker", "push", imageName+":latest"); err != nil {
		return err
	}
	return nil
}

// Builds docker image
func (Docker) Build(ctx context.Context) {
	mg.Deps(
		Clean,
		Test.Verbose,
		pluginCheckCmd,
		pluginCheck2CmdLinux,
		buildDockerImage)
}

// Build and push docker image
func (Docker) Push(ctx context.Context) {
	mg.Deps(
		Clean,
		Test.Verbose,
		pluginCheckCmd,
		pluginCheck2CmdLinux,
		buildDockerImage,
		pushDockerImage)
}

// deprecated - use pluginCheck2
func pluginCheckCmd() error {
	os.Setenv("GO111MODULE", "on")
	os.Setenv("CGO_ENABLED", "0")
	return sh.RunV("go", "build", "-o", "bin/plugincheck", "./pkg/cmd/plugincheck")
}

func buildCommand(command string, arch string) error {
	env, ok := archTargets[arch]
	if !ok {
		return fmt.Errorf("unknown arch %s", arch)
	}
	log.Printf("Building %s/%s\n", arch, command)
	outDir := fmt.Sprintf("./bin/%s/%s", arch, command)
	cmdDir := fmt.Sprintf("./pkg/cmd/%s", command)
	if err := sh.RunWith(env, "go", "build", "-o", outDir, cmdDir); err != nil {
		return err
	}

	// intentionally igores errors
	sh.RunV("chmod", "+x", outDir)
	return nil
}

// build all commands inside ./pkg/cmd for current architecture
func (Build) Commands(ctx context.Context) error {
	mg.Deps(
		Clean,
	)

	const commandsFolder = "./pkg/cmd"
	folders, err := ioutil.ReadDir(commandsFolder)

	if err != nil {
		return err
	}

	for _, folder := range folders {
		if folder.IsDir() {
			currentArch := runtime.GOOS + "_" + runtime.GOARCH
			err := buildCommand(folder.Name(), currentArch)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func pluginCheck2CmdDarwin() error {
	return buildCommand("plugincheck2", "darwin_amd64")
}
func pluginCheck2CmdLinux() error {
	return buildCommand("plugincheck2", "linux_amd64")
}

func testVerbose() error {
	os.Setenv("GO111MODULE", "on")
	os.Setenv("CGO_ENABLED", "0")
	return sh.RunV("go", "test", "-v", "./pkg/...")
}

func test() error {
	os.Setenv("GO111MODULE", "on")
	os.Setenv("CGO_ENABLED", "0")
	return sh.RunV("go", "test", "./pkg/...")
}

// Formats the source files
func (Build) Format() error {
	if err := sh.RunV("gofmt", "-w", "./pkg"); err != nil {
		return err
	}
	return nil
}

// Minimal build
func (Build) Local(ctx context.Context) {
	mg.Deps(
		Clean,
		pluginCheckCmd,
		Build.Commands,
	)
}

// Lint/Format/Test/Build
func (Build) CI(ctx context.Context) {
	mg.Deps(
		Build.Lint,
		Build.Format,
		Test.Verbose,
		Clean,
		pluginCheck2CmdLinux,
	)
}

func (Build) All(ctx context.Context) {
	mg.Deps(
		Build.Lint,
		Build.Format,
		Test.Verbose,
		pluginCheck2CmdLinux,
		pluginCheck2CmdDarwin,
	)

}

// Run linter against codebase
func (Build) Lint() error {
	os.Setenv("GO111MODULE", "on")
	log.Printf("Linting...")
	return sh.RunV("golangci-lint", "--timeout", "3m", "run", "-v", "./pkg/...")
}

// Run tests in verbose mode
func (Test) Verbose() {
	mg.SerialDeps(
		Build.Local,
		testVerbose,
	)
}

// Run tests in normal mode
func (Test) Default() {
	mg.SerialDeps(
		Build.Local,
		test,
	)
}

// Integration runs the integration tests for the plugin validator
func (Test) Integration() error {
	mg.Deps(Build.Local)
	return sh.RunV("go", "test", "-v", "./pkg/cmd/plugincheck2")
}

// Removes built files
func Clean() {
	log.Printf("Cleaning all")
	os.RemoveAll("./bin/plugincheck")
	os.RemoveAll("./bin/linux_amd64")
	os.RemoveAll("./bin/darwin_amd64")
	os.RemoveAll("./bin/darwin_arm64")
}

// Build and Run V1
func (Run) V1() error {
	mg.Deps(Build.Local)
	return sh.RunV(
		"./bin/plugincheck",
		"https://github.com/marcusolsson/grafana-jsonapi-datasource/releases/download/v0.6.0/marcusolsson-json-datasource-0.6.0.zip",
	)
}

// Build and Run V2
func (Run) V2() error {
	mg.Deps(Build.Local)
	return sh.RunV(
		"./bin/"+runtime.GOOS+"_"+runtime.GOARCH+"/plugincheck2",
		"-config",
		"config/verbose-json.yaml",
		"https://github.com/marcusolsson/grafana-jsonapi-datasource/releases/download/v0.6.0/marcusolsson-json-datasource-0.6.0.zip",
	)
}

func (Run) V2Local(ctx context.Context, path string, sourceCodePath string) error {
	mg.Deps(Build.Local)

	configFile := "config/pipeline.yaml"
	// if config/custom.yaml exists, use it instead
	if _, err := os.Stat("config/custom.yaml"); err == nil {
		configFile = "config/custom.yaml"
	}

	if !strings.HasPrefix(sourceCodePath, "http://") &&
		!strings.HasPrefix(sourceCodePath, "https://") &&
		!strings.HasPrefix(sourceCodePath, "file://") {
		sourceCodePath = "file://" + sourceCodePath
	}

	command := []string{
		"./bin/" + runtime.GOOS + "_" + runtime.GOARCH + "/plugincheck2",
		"-config",
		configFile,
	}
	if sourceCodePath != "" {
		command = append(command, "-sourceCodeUri", sourceCodePath)
	}
	command = append(command, path)

	return sh.RunWithV(map[string]string{
		"SKIP_CLAMAV": "1",
	}, command[0], command[1:]...)
}

func (Run) SourceDiffLocal(ctx context.Context, archive string, source string) error {
	buildCommand("sourcemapdiff", runtime.GOOS+"_"+runtime.GOARCH)

	// if source doesn't start with http or file:// add file://
	if !strings.HasPrefix(source, "http://") &&
		!strings.HasPrefix(source, "file://") &&
		!strings.HasPrefix(source, "https://") {
		source = "file://" + source
	}

	command := []string{
		"./bin/" + runtime.GOOS + "_" + runtime.GOARCH + "/sourcemapdiff",
		"-archiveUri", archive}

	if source != "" {
		command = append(command, "-sourceCodeUri", source)
	}

	return sh.RunV(command[0], command[1:]...)

}

// Readme re-generates the "Analyzers" table in README.md.
func (Gen) Readme() error {
	return sh.RunV("go", "test", "-v", "./pkg/genreadme", "-generate")
}
