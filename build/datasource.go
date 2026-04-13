package build

import (
	"fmt"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema"
	"github.com/magefile/mage/mg"
)

// Datasource datasource extraction commands.
type Datasource mg.Namespace

// GenerateOpenAPI generates datasource OpenAPI extension JSON and outputs to stdout.
func (Datasource) GenerateOpenAPI(dir string) error {
	if dir == "" {
		dir = "."
	}
	if os.Getenv("GOTOOLCHAIN") == "" {
		if err := os.Setenv("GOTOOLCHAIN", "auto"); err != nil {
			return err
		}
	}

	result, err := datasourceschema.GenerateOpenAPI(datasourceschema.OpenAPIOptions{
		Dir: dir,
	})
	if err != nil {
		return err
	}

	return writeDocument(result.Body, result.Warnings)
}

// GenerateQueryTypes generates datasource query type definitions JSON and outputs to stdout.
func (Datasource) GenerateQueryTypes(dir string) error {
	if dir == "" {
		dir = "."
	}
	if os.Getenv("GOTOOLCHAIN") == "" {
		if err := os.Setenv("GOTOOLCHAIN", "auto"); err != nil {
			return err
		}
	}

	result, err := datasourceschema.GenerateQueryTypes(datasourceschema.OpenAPIOptions{
		Dir: dir,
	})
	if err != nil {
		return err
	}

	return writeDocument(result.Body, result.Warnings)
}

func writeOpenAPIWarnings(f *os.File, warnings []datasourceschema.OpenAPIWarning) error {
	for _, warning := range warnings {
		if _, err := fmt.Fprintf(f, "warning: %s: %s", warning.Code, warning.Message); err != nil {
			return err
		}
		if warning.Position.File != "" {
			if _, err := fmt.Fprintf(f, " (%s:%d:%d)", warning.Position.File, warning.Position.Line, warning.Position.Column); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(f); err != nil {
			return err
		}
	}

	return nil
}

func writeDocument(body []byte, warnings []datasourceschema.OpenAPIWarning) error {
	if err := writeOpenAPIWarnings(os.Stderr, warnings); err != nil {
		return err
	}

	if _, err := os.Stdout.Write(body); err != nil {
		return err
	}
	_, err := fmt.Fprintln(os.Stdout)
	return err
}
