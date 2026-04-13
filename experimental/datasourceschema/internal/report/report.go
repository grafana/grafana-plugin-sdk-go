package report

import (
	"fmt"
	"sort"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/datasourceschema/internal/model"
)

func Merge(base model.Report, extras ...model.Report) model.Report {
	merged := model.Report{
		Findings: append([]model.Finding{}, base.Findings...),
		Warnings: append([]model.Warning{}, base.Warnings...),
	}

	for _, extra := range extras {
		merged.Findings = append(merged.Findings, extra.Findings...)
		merged.Warnings = append(merged.Warnings, extra.Warnings...)
	}

	return Normalize(merged)
}

func Normalize(in model.Report) model.Report {
	out := model.Report{}

	seenFindings := map[string]struct{}{}
	for _, finding := range in.Findings {
		key := findingKey(finding)
		if _, ok := seenFindings[key]; ok {
			continue
		}
		seenFindings[key] = struct{}{}
		out.Findings = append(out.Findings, finding)
	}

	seenWarnings := map[string]struct{}{}
	for _, warning := range in.Warnings {
		key := warningKey(warning)
		if _, ok := seenWarnings[key]; ok {
			continue
		}
		seenWarnings[key] = struct{}{}
		out.Warnings = append(out.Warnings, warning)
	}

	sort.Slice(out.Findings, func(i int, j int) bool {
		left := out.Findings[i]
		right := out.Findings[j]
		if left.Position.File != right.Position.File {
			return left.Position.File < right.Position.File
		}
		if left.Position.Line != right.Position.Line {
			return left.Position.Line < right.Position.Line
		}
		return left.Kind < right.Kind
	})

	sort.Slice(out.Warnings, func(i int, j int) bool {
		left := out.Warnings[i]
		right := out.Warnings[j]
		if left.Position.File != right.Position.File {
			return left.Position.File < right.Position.File
		}
		if left.Position.Line != right.Position.Line {
			return left.Position.Line < right.Position.Line
		}
		return left.Code < right.Code
	})

	return out
}

func findingKey(f model.Finding) string {
	target := ""
	if f.Target != nil {
		target = fmt.Sprintf("%s.%s.%t.%s", f.Target.PackagePath, f.Target.TypeName, f.Target.Pointer, f.Target.TypeString)
		if f.Target.Expr != nil {
			target += fmt.Sprintf(".%s:%d:%d", f.Target.Expr.File, f.Target.Expr.Line, f.Target.Expr.Column)
		}
	}

	return fmt.Sprintf(
		"%s|%s|%s|%d|%s|%s|%s|%s",
		f.Kind,
		f.Source,
		f.Position.File,
		f.Position.Line,
		target,
		f.Key,
		f.Pattern,
		f.Destination,
	)
}

func warningKey(w model.Warning) string {
	return fmt.Sprintf("%s|%s|%d|%s", w.Code, w.Position.File, w.Position.Line, w.Message)
}
