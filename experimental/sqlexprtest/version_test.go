package sqlexprtest

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana-plugin-sdk-go/data"
)

// withFix pretends a Grafana release v fixed the behavior a check reports,
// or, when v is empty, that no release fixed it yet.
func withFix(t *testing.T, code Code, v string) {
	t.Helper()
	prev, had := grafanaFixes[code]
	if v == "" {
		delete(grafanaFixes, code)
	} else {
		grafanaFixes[code] = v
	}
	t.Cleanup(func() {
		if had {
			grafanaFixes[code] = prev
		} else {
			delete(grafanaFixes, code)
		}
	})
}

func TestParseVersion(t *testing.T) {
	for in, want := range map[string]version{
		"13.3.0":     {13, 3, 0},
		"v12.1.4":    {12, 1, 4},
		"13.3.0-pre": {13, 3, 0},
		"11.0.0+abc": {11, 0, 0},
		"12":         {12, 0, 0},
		"12.4":       {12, 4, 0},
	} {
		got, err := parseVersion(in)
		require.NoError(t, err, in)
		assert.Equal(t, want, got, in)
	}
	for _, in := range []string{"", "x", "1.2.3.4", "1.-2"} {
		_, err := parseVersion(in)
		assert.Error(t, err, in)
	}
}

func TestLowestVersion(t *testing.T) {
	for in, want := range map[string]string{
		">=12.3.0":                  "12.3.0",
		">=10.4.0 <12":              "10.4.0",
		"^11.0.0":                   "11.0.0",
		">=12.1.0 || >=10.4.0 <11":  "10.4.0",
		">=11.6.0-0":                "11.6.0",
		"10.4.0":                    "10.4.0",
		" >= 12.0.0":                "12.0.0",
		">= 10.4.0 < 12":            "10.4.0",
		"<12.0.0":                   "",
		">=10.4.0 <11 || >=12.1.0 ": "10.4.0",
	} {
		got, err := lowestVersion(in)
		if want == "" {
			assert.Error(t, err, in)
			continue
		}
		require.NoError(t, err, in)
		assert.Equal(t, want, got.String(), in)
	}
}

func TestMinGrafanaPanicsOnInvalidVersion(t *testing.T) {
	assert.Panics(t, func() { MinGrafana("latest") })
}

func TestMinGrafanaFromPluginJSON(t *testing.T) {
	write := func(t *testing.T, content string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "plugin.json")
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
		return path
	}

	t.Run("reads the lower bound", func(t *testing.T) {
		opt, err := MinGrafanaFromPluginJSON(write(t, `{"dependencies": {"grafanaDependency": ">=12.3.0"}}`))
		require.NoError(t, err)
		assert.Equal(t, "12.3.0", newOptions([]Option{opt}).minGrafana.String())
	})

	t.Run("errors", func(t *testing.T) {
		for _, content := range []string{`{}`, `{"dependencies": {"grafanaDependency": "<12"}}`, `not json`} {
			_, err := MinGrafanaFromPluginJSON(write(t, content))
			assert.Error(t, err, content)
		}
		_, err := MinGrafanaFromPluginJSON(filepath.Join(t.TempDir(), "missing.json"))
		assert.Error(t, err)
	})
}

func clashFrames() data.Frames {
	return data.Frames{typed(data.FrameTypeTimeSeriesMulti, data.NewFrame("",
		data.NewField("time", nil, []time.Time{t0}),
		data.NewField("cpu", data.Labels{"time": "nightly", "label_time": "x"}, []float64{1}),
	))}
}

func TestCheckGeneratedNameClashFixedIn(t *testing.T) {
	t.Run("no fix yet", func(t *testing.T) {
		withFix(t, CodeGeneratedNameClash, "")
		r := Check(clashFrames(), MinGrafana("99.0.0"))
		require.Equal(t, []Code{CodeGeneratedNameClash}, codes(r))
		assert.Equal(t, SeverityWarning, r.Findings[0].Severity)
		assert.Empty(t, r.Findings[0].FixedIn)
		assert.Equal(t, []string{"time", "__value__", "__metric_name__", "label_time", "time"}, columnNames(r))
	})

	withFix(t, CodeGeneratedNameClash, "13.3.0")

	for _, tc := range []struct {
		name string
		opts []Option
	}{
		{"no minimum version", nil},
		{"older minimum version", []Option{MinGrafana("13.2.9")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := Check(clashFrames(), tc.opts...)
			require.Equal(t, []Code{CodeGeneratedNameClash}, codes(r))
			f := r.Findings[0]
			assert.Equal(t, SeverityWarning, f.Severity)
			assert.Equal(t, "13.3.0", f.FixedIn)
			assert.Contains(t, f.Message, `Grafana 13.3.0 and later name the column "label_label_time"`)
			assert.Contains(t, f.String(), "fixed in Grafana 13.3.0")
			assert.Equal(t, []string{"time", "__value__", "__metric_name__", "label_time", "time"}, columnNames(r))
		})
	}

	for _, minimum := range []string{"13.3.0", "13.3.0-pre", "14.0.0"} {
		t.Run("fixed in every supported version "+minimum, func(t *testing.T) {
			r := Check(clashFrames(), MinGrafana(minimum))
			require.Equal(t, []Code{CodeGeneratedNameClash}, codes(r))
			f := r.Findings[0]
			assert.Equal(t, SeverityInfo, f.Severity)
			assert.Equal(t, "13.3.0", f.FixedIn)
			// "label_time" is taken by the other label, so Grafana adds the prefix again.
			assert.Equal(t, []string{"time", "__value__", "__metric_name__", "label_time", "label_label_time"}, columnNames(r))
		})
	}
}

func TestCheckGeneratedNameClashOnlyWithColumnsGrafanaGenerates(t *testing.T) {
	// Numeric frames have no time column, and frames without a display name
	// have no __display_name__ column, so these labels clash with nothing.
	r := Check(data.Frames{typed(data.FrameTypeNumericWide, data.NewFrame("",
		data.NewField("cpu", data.Labels{"time": "nightly", "__display_name__": "x"}, []float64{1}),
	))})
	assert.Empty(t, codes(r))
	assert.Equal(t, []string{"__metric_name__", "__value__", "__display_name__", "time"}, columnNames(r))
}

func TestAssertUsesMinGrafana(t *testing.T) {
	withFix(t, CodeGeneratedNameClash, "13.3.0")

	rec := &recorder{TB: t}
	AssertFrames(rec, clashFrames())
	assert.Len(t, rec.errors, 1, "the plugin supports versions without the fix")

	rec = &recorder{TB: t}
	AssertFrames(rec, clashFrames(), MinGrafana("13.3.0"))
	assert.Empty(t, rec.errors, "every supported version has the fix, so the finding is advice")
}
