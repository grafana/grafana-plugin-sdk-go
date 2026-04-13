package datasourceschema

import (
	"encoding/json"

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

func GenerateOpenAPI(opts OpenAPIOptions) (*OpenAPIResult, error) {
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
		GenerateSpec:    true,
		GenerateQueries: true,
	})
	if err != nil {
		return nil, err
	}

	body, err := json.MarshalIndent(result.Extension, "", "  ")
	if err != nil {
		return nil, err
	}

	return &OpenAPIResult{
		Body:     body,
		Warnings: warningsFromModel(result.Warnings),
	}, nil
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
