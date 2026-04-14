//go:build mage

package main

import (
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Build builds the binaries.
func Build() error {
	return sh.RunV("go", "build", "./...")
}

// DataGenerate runs go generate (code generation) on the data package.
func DataGenerate() error {
	return sh.Run("go", "generate", "./data")
}

// Protobuf protobuf related commands.
type Protobuf mg.Namespace

// Generate generates protobuf files.
func (Protobuf) Generate() error {
	return sh.RunV("buf", "generate", "proto", "--template", "./proto/buf.gen.yaml")
}

// Validate validate breaking changes in protobuf files.
func (Protobuf) Validate() error {
	return sh.RunV("buf", "breaking", "proto", "--against", "https://github.com/grafana/grafana-plugin-sdk-go.git#branch=main,subdir=proto")
}

// Test runs the test suite.
func Test() error {
	return sh.RunV("go", "test", "./...")
}

// TestRace runs the test suite with the data race detector enabled.
func TestRace() error {
	return sh.RunV("go", "test", "-race", "./...")
}

func Lint() error {
	if err := sh.RunV("golangci-lint", "run", "./..."); err != nil {
		return err
	}

	return nil
}

// Drone signs the Drone configuration file
// This needs to be run everytime the drone.yml file is modified
// See https://github.com/grafana/deployment_tools/blob/master/docs/infrastructure/drone/signing.md for more info
func Drone() error {
	if err := sh.RunV("drone", "lint"); err != nil {
		return err
	}

	if err := sh.RunV("drone", "--server", "https://drone.grafana.net", "sign", "--save", "grafana/grafana-plugin-sdk-go"); err != nil {
		return err
	}

	return nil
}

var Aliases = map[string]interface{}{
	"protobuf": Protobuf.Generate,
}

var Default = Build
