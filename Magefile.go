//+build mage

package main

import (
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

// Protobuf generates protobuf files.
func Protobuf() error {
	if err := sh.RunV("./scripts/protobuf-check.sh"); err != nil {
		return err
	}

	return sh.RunV("./proto/generate.sh")
}

// Test runs the test suite.
func Test() error {
	return sh.RunV("go", "test", "./...")
}

func Lint() error {
	if err := sh.RunV("golangci-lint", "run", "./..."); err != nil {
		return err
	}
	if err := sh.RunV("revive", "-formatter", "stylish", "-config", "scripts/configs/revive.toml", "./..."); err != nil {
		return err
	}

	return nil
}

var Default = Build
