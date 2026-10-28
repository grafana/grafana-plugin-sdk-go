package sqlexprtest

import (
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/data"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionPrecedence(t *testing.T) {
	ordered := []string{"13.3.0-0", "13.3.0-alpha", "13.3.0-alpha.1", "13.3.0-alpha.2", "13.3.0-alpha.10", "13.3.0-alpha.beta", "13.3.0-beta", "13.3.0-beta.2", "13.3.0-beta.11", "13.3.0-rc.1", "13.3.0", "13.3.1", "13.4.0", "14.0.0"}
	for i, a := range ordered {
		v, err := parseVersion(a)
		require.NoError(t, err)
		assert.Equal(t, a, v.String())
		for j, b := range ordered {
			other, err := parseVersion(b)
			require.NoError(t, err)
			assert.Equal(t, i < j, v.less(other), "%s < %s", a, b)
		}
	}
	a, err := parseVersion("13.3.0+build.01")
	require.NoError(t, err)
	b, err := parseVersion("v13.3.0+other")
	require.NoError(t, err)
	assert.Equal(t, a, b)
	for _, input := range []string{"01.2.3", "1.02.3", "1.2.03", "1.2.3-01", "1.2.3-alpha.01", "1.2.3-a..b", "1.2.3+", "1.2.3+build..1", "1.2-pre", "1.2.3-", "1.2.3.4", "-1.2.3", "1.2.99999999999999999999999"} {
		_, err := parseVersion(input)
		assert.Error(t, err, input)
	}
}

func TestDependencyMinimums(t *testing.T) {
	for input, expected := range map[string]string{
		">=12.0.0 >=13.3.0":        "13.3.0",
		"<12 || >=13.3.0":          "0.0.0",
		">13.2.9":                  "13.2.10",
		">13.2":                    "13.3.0",
		">13":                      "14.0.0",
		"13":                       "13.0.0",
		"=13.2":                    "13.2.0",
		"<=13":                     "0.0.0",
		">=13.3.0 <12 || >=12.0.0": "12.0.0",
		">=13.3.0 <=13.3.0":        "13.3.0",
		"^13.2.1 >=13.3.0":         "13.3.0",
		"~13.2.1 >=13.2.2":         "13.2.2",
		"^0.2.1 >=0.2.2":           "0.2.2",
		"^0.0 >=0.0.7":             "0.0.7",
		"^0 >=0.9.0":               "0.9.0",
		"~13 >=13.2.0":             "13.2.0",
		">=13.3.0-pre <14":         "13.3.0-pre",
		">13.3.0-alpha":            "13.3.0-alpha.0",
		">13.2.9 <13.2.10-beta":    "13.2.10-0",
		">=13.3.0-pre >=13.3.0":    "13.3.0",
		">=13.3.0-pre <13.3.0":     "13.3.0-pre",
		">=13.2.0 <13.2.1-beta":    "13.2.0",
		"^13.2.0-beta.1":           "13.2.0-beta.1",
		"13.3.0-pre+build.1":       "13.3.0-pre",
	} {
		t.Run(input, func(t *testing.T) {
			v, err := lowestVersion(input)
			require.NoError(t, err)
			assert.Equal(t, expected, v.String())
		})
	}
}

func TestDependencyRejectsEmptyOrUnsupportedRanges(t *testing.T) {
	for _, input := range []string{"", " ", "|| >=13", ">=13 ||", ">=13 || || >=14", "13.x", "*", "13.0.0 - 14.0.0", ">=13, <14", "!=13.0.0", ">=13.3.0 <13.3.0", ">13.3.0 <=13.3.0", "13.0.0 14.0.0", "^13.2.0 >=14.0.0", "~13.2.1 >=13.3.0", "^0.0.1 >=0.0.2", "^13.0.0 >=14.0.0-beta", ">=13.0.0 <13", "^0.2.1 >=0.3.0", ">=13.0.0 || 14.x"} {
		t.Run(input, func(t *testing.T) {
			_, err := lowestVersion(input)
			require.Error(t, err)
		})
	}
}

func TestKnownFutureFixPreservesCurrentBehavior(t *testing.T) {
	withFix(t, "99.0.0")
	r := Check(clashFrames(), MinGrafana("13.3.0-pre"))
	require.Len(t, r.Findings, 1)
	assert.Equal(t, "99.0.0", r.Findings[0].FixedIn)
	assert.Equal(t, SeverityWarning, r.Findings[0].Severity)
	assert.Equal(t, []string{"time", "__value__", "__metric_name__", "label_time", "time"}, columnNames(r))
}

func TestAppliedFixChecksFinalColumnNames(t *testing.T) {
	withFix(t, "13.3.0")
	frames := data.Frames{typed(data.FrameTypeTimeSeriesMulti, data.NewFrame("",
		data.NewField("time", nil, []time.Time{t0}),
		data.NewField("v", data.Labels{"TIME": "a", "time": "b", "label_TIME": "c", "label_label_time": "d"}, []float64{1}),
	))}
	old := Check(frames, MinGrafana("13.3.0-pre"))
	assert.True(t, old.Has(CodeDuplicateName))
	r := Check(frames, MinGrafana("13.3.0"))
	assert.False(t, r.Has(CodeDuplicateName))
	assert.Equal(t, SeverityInfo, r.Worst())
	assert.Equal(t, []string{"time", "__value__", "__metric_name__", "label_label_label_TIME", "label_TIME", "label_label_time", "label_label_label_label_time"}, columnNames(r))
}

func TestDisplayColumnCollisionIsVersioned(t *testing.T) {
	withFix(t, "13.3.0")
	f := data.NewField("v", data.Labels{"__display_name__": "label"}, []float64{1})
	f.Config = &data.FieldConfig{DisplayNameFromDS: "display"}
	frames := data.Frames{typed(data.FrameTypeNumericWide, data.NewFrame("", f))}
	old := Check(frames, MinGrafana("13.2.9"))
	current := Check(frames, MinGrafana("13.3.0"))
	assert.Equal(t, SeverityWarning, old.Worst())
	assert.Equal(t, SeverityInfo, current.Worst())
	assert.Equal(t, []string{"__metric_name__", "__value__", "__display_name__", "label___display_name__"}, columnNames(current))
}
