//go:build mage
// +build mage

package main

import (
	"context"
	"log"
	"os"
	"runtime"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	// mg contains helpful utility functions, like Deps
)

type Build mg.Namespace
type Test mg.Namespace
type Docker mg.Namespace
type Run mg.Namespace

const imageName = "grafana/plugin-validator-cli"
const imageVersion = "v2"

// Default target to run when none is specified
// If not set, running mage will list available targets
var Default = Build.Local

/* Docker */
func buildDockerImage() error {
	return sh.RunV("docker", "build", "-t", imageName+":"+imageVersion, "-t", imageName+":latest", "-f", "Dockerfile", ".")
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

/* executables */
func pluginCheckCmd() error {
	os.Setenv("GO111MODULE", "on")
	os.Setenv("CGO_ENABLED", "0")
	return sh.RunV("go", "build", "-o", "bin/plugincheck", "./pkg/cmd/plugincheck")
}

func pluginCheck2CmdDarwin() error {
	env := map[string]string{
		"CGO_ENABLED": "0",
		"GO111MODULE": "on",
		"GOARCH":      "amd64",
		"GOOS":        "darwin",
	}
	if err := sh.RunWith(env, "go", "build", "-o", "bin/darwin_amd64/plugincheck2", "./pkg/cmd/plugincheck2"); err != nil {
		return err
	}
	return nil
}
func pluginCheck2CmdLinux() error {
	env := map[string]string{
		"CGO_ENABLED": "0",
		"GO111MODULE": "on",
		"GOARCH":      "amd64",
		"GOOS":        "linux",
	}
	if err := sh.RunWith(env, "go", "build", "-o", "bin/linux_amd64/plugincheck2", "./pkg/cmd/plugincheck2"); err != nil {
		return err
	}
	return nil
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
		pluginCheck2CmdDarwin,
		pluginCheck2CmdLinux,
	)
}

// Lint/Format/Test/Build
func (Build) CI(ctx context.Context) {
	mg.Deps(
		Build.Lint,
		Build.Format,
		Test.Verbose,
		Clean,
		pluginCheckCmd,
		pluginCheck2CmdDarwin,
		pluginCheck2CmdLinux,
	)
}

// Run linter against codebase
func (Build) Lint() error {
	os.Setenv("GO111MODULE", "on")
	log.Printf("Linting...")
	return sh.RunV("golangci-lint", "run", "./pkg/...")
}

// Run tests in verbose mode
func (Test) Verbose() {
	mg.Deps(
		testVerbose,
	)
}

// Run tests in normal mode
func (Test) Default() {
	mg.Deps(
		test,
	)
}

// Removes built files
func Clean() error {
	log.Printf("Cleaning all")
	os.RemoveAll("./bin/plugincheck")
	os.RemoveAll("./bin/linux_amd64")
	return os.RemoveAll("./bin/darwin_amd64")
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

func (Run) V2Local(ctx context.Context, path string) error {
	mg.Deps(Build.Local)

	if _, err := os.Stat(path); err != nil {
		return err
	}

	return sh.RunV(
		"./bin/"+runtime.GOOS+"_"+runtime.GOARCH+"/plugincheck2",
		"-config",
		"config/pipeline.yaml",
		path)
}
