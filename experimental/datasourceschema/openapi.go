package datasourceschema

import (
	"encoding/json"

	v0alpha1 "github.com/grafana/grafana-plugin-sdk-go/experimental/apis/datasource/v0alpha1"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/analyze"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/openapigen"
)

type OpenAPIOptions struct {
	Dir        string
	Patterns   []string
	BuildFlags []string
}

type OpenAPIPosition struct {
	File   string
	Line   int
	Column int
}

type OpenAPIWarning struct {
	Position OpenAPIPosition
	Code     string
	Message  string
}

type OpenAPIResult struct {
	Body     []byte
	Warnings []OpenAPIWarning
}

type QueryTypesResult struct {
	Body     []byte
	Warnings []OpenAPIWarning
}

func GenerateOpenAPI(opts OpenAPIOptions) (*OpenAPIResult, error) {
	result, err := generate(opts, true, false)
	if err != nil {
		return nil, err
	}

	body, err := json.MarshalIndent(result.OpenAPI, "", "  ")
	if err != nil {
		return nil, err
	}

	return &OpenAPIResult{
		Body:     body,
		Warnings: warningsFromModel(result.Warnings),
	}, nil
}

func GenerateQueryTypes(opts OpenAPIOptions) (*QueryTypesResult, error) {
	result, err := generate(opts, false, true)
	if err != nil {
		return nil, err
	}

	body, err := marshalQueryTypes(result.QueryTypes)
	if err != nil {
		return nil, err
	}

	return &QueryTypesResult{
		Body:     body,
		Warnings: warningsFromModel(result.Warnings),
	}, nil
}

func generate(opts OpenAPIOptions, generateSpec bool, generateQueries bool) (*openapigen.Result, error) {
	cfg := normalizeOptions(opts)

	report, err := analyze.Run(analyze.Config{
		Dir:        cfg.Dir,
		Patterns:   cfg.Patterns,
		BuildFlags: cfg.BuildFlags,
		UseSSA:     true,
	})
	if err != nil {
		return nil, err
	}

	result, err := openapigen.Build(openapigen.Options{
		Dir:             cfg.Dir,
		Patterns:        cfg.Patterns,
		BuildFlags:      cfg.BuildFlags,
		Report:          *report,
		GenerateSpec:    generateSpec,
		GenerateQueries: generateQueries,
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func marshalQueryTypes(queryTypes *v0alpha1.QueryTypeDefinitionList) ([]byte, error) {
	if queryTypes == nil {
		queryTypes = &v0alpha1.QueryTypeDefinitionList{}
	}
	return json.MarshalIndent(queryTypes, "", "  ")
}

func normalizeOptions(opts OpenAPIOptions) OpenAPIOptions {
	cfg := opts
	if cfg.Dir == "" {
		cfg.Dir = "."
	}
	if len(cfg.Patterns) == 0 {
		cfg.Patterns = []string{"./..."}
	}
	return cfg
}

func warningsFromModel(in []model.Warning) []OpenAPIWarning {
	out := make([]OpenAPIWarning, 0, len(in))
	for _, warning := range in {
		out = append(out, OpenAPIWarning{
			Position: OpenAPIPosition{
				File:   warning.Position.File,
				Line:   warning.Position.Line,
				Column: warning.Position.Column,
			},
			Code:    warning.Code,
			Message: warning.Message,
		})
	}
	return out
}
