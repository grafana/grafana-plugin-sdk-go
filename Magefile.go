//go:build mage
// +build mage

package main

import (
	"fmt"
	"github.com/magefile/mage/sh"
	"runtime"
)

// Build builds the binaries.
func Build() error {
	return sh.RunV("go", "build", "./...")
}

// DataGenerate runs go generate (code generation) on the data package.
func DataGenerate() error {
	return sh.Run("go", "generate", "./data")
}

// Protobuf generates protobuf files.
func Protobuf() error {
	if err := sh.RunV("./scripts/protobuf-check.sh"); err != nil {
		return err
	}

	return sh.RunV("./proto/generate.sh")
}

// Info writes system info
func Info() error {
	fmt.Printf("runtime.GOOS: %s\n", runtime.GOOS)
	fmt.Printf("runtime.GOARCH: %s\n", runtime.GOARCH)
	return nil
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

var Default = Build
