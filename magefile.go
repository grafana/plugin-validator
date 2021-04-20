// +build mage

package main

import (
	"context"
	"log"
	"os"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	// mg contains helpful utility functions, like Deps
)

type Build mg.Namespace
type Test mg.Namespace
type Docker mg.Namespace

// Default target to run when none is specified
// If not set, running mage will list available targets
var Default = Build.Cmds

func pluginCheckCmd() error {
	os.Setenv("GO111MODULE", "on")
	log.Printf("Building...")
	return sh.RunV("go", "build", "-o", "bin/plugincheck", "./pkg/cmd/plugincheck")
}

func pluginCheck2Cmd() error {
	os.Setenv("GO111MODULE", "on")
	log.Printf("Building...")
	return sh.RunV("go", "build", "-o", "bin/plugincheck2", "./pkg/cmd/plugincheck2")
}

func (Build) Cmds(ctx context.Context) {
	mg.Deps(
		Clean,
		pluginCheckCmd,
		pluginCheck2Cmd)
}

func (Build) Local(ctx context.Context) {
	mg.Deps(
		Clean,
		pluginCheckCmd,
		pluginCheck2Cmd)
}

func (Build) CI(ctx context.Context) {
	mg.Deps(
		Build.Lint,
		Test.Verbose,
		Clean,
		pluginCheckCmd,
		pluginCheck2Cmd)
}

// Run linter against codebase
func (Build) Lint() error {
	os.Setenv("GO111MODULE", "on")
	log.Printf("Linting...")
	return sh.RunV("golangci-lint", "run", "./pkg/...")
}

func testVerbose() error {
	os.Setenv("GO111MODULE", "on")
	log.Printf("Testing...")
	return sh.RunV("go", "test", "-v", "./pkg/...")
}

func test() error {
	os.Setenv("GO111MODULE", "on")
	log.Printf("Testing...")
	return sh.RunV("go", "test", "./pkg/...")
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

// Clean removes built files
func Clean() error {
	log.Printf("Cleaning all")
	os.RemoveAll("./bin/plugincheck")
	return os.RemoveAll("./bin/plugincheck2")
}

// Build and Run Application
func RunV1() error {
	mg.Deps(Build.Cmds)
	return sh.RunV("./bin/plugincheck", "-c", "config/verbose.yaml")
}

// Build and Run Application
func RunV2() error {
	mg.Deps(Build.Cmds)
	return sh.RunV("./bin/plugincheck2", "-c", "config/verbose.yaml")
}
