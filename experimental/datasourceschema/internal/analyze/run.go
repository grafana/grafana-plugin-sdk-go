package analyze

import (
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/load"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/patterns"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/report"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/ssaresolve"
)

type Config struct {
	Dir        string
	Patterns   []string
	BuildFlags []string
	UseSSA     bool
}

func Run(cfg Config) (*model.Report, error) {
	loadRes, err := load.Packages(load.Config{
		Dir:        cfg.Dir,
		Patterns:   cfg.Patterns,
		BuildFlags: cfg.BuildFlags,
		NeedModule: true,
	})
	if err != nil {
		return nil, err
	}

	typedRes, err := patterns.RunTyped(patterns.NewContext(loadRes))
	if err != nil {
		return nil, err
	}

	finalReport := typedRes.Report
	if !cfg.UseSSA {
		normalized := report.Normalize(finalReport)
		return &normalized, nil
	}

	resolver, err := ssaresolve.Build(loadRes)
	if err != nil {
		normalized := report.Normalize(finalReport)
		return &normalized, nil
	}

	var findings []model.Finding
	var warnings []model.Warning
	if len(typedRes.Pending) > 0 {
		findings, warnings, err = resolver.Resolve(typedRes.Pending)
		if err != nil {
			normalized := report.Normalize(finalReport)
			return &normalized, nil
		}
	}

	merged := report.Merge(finalReport, model.Report{
		Findings: findings,
		Warnings: warnings,
	})
	if !hasQueryFinding(merged.Findings) {
		localFindings, localWarnings := resolver.InferLocalQueryTargets()
		merged = report.Merge(merged, model.Report{
			Findings: localFindings,
			Warnings: localWarnings,
		})
	}
	if !hasQueryFinding(merged.Findings) {
		frameworkFindings, frameworkWarnings := resolver.InferFrameworkQueryTargets()
		merged = report.Merge(merged, model.Report{
			Findings: frameworkFindings,
			Warnings: frameworkWarnings,
		})
	}

	normalized := report.Normalize(merged)
	return &normalized, nil
}

func hasQueryFinding(findings []model.Finding) bool {
	for _, finding := range findings {
		if finding.Source == model.SourceKindQueryJSON && finding.Target != nil {
			return true
		}
	}
	return false
}
