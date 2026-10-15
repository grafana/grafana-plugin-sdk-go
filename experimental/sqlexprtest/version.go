package sqlexprtest

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// grafanaFixes maps a check to the first Grafana version that no longer has
// the behavior it reports. Add an entry when a Grafana release fixes the
// behavior, and model the fixed behavior in the check. When every supported
// version has the fix, the check still reports a SeverityInfo finding with
// FixedIn set, so the report says what changed. Never remove the check while
// plugins still support older versions.
var grafanaFixes = map[Code]string{}

// version is a Grafana release version. Check ignores pre-release and build
// suffixes, so 13.3.0-pre counts as 13.3.0, the release it becomes.
type version struct {
	major, minor, patch int
}

func parseVersion(s string) (version, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "v")
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return version{}, fmt.Errorf("invalid Grafana version %q", s)
	}
	var n [3]int
	for i, p := range parts {
		v, err := strconv.Atoi(p)
		if err != nil || v < 0 {
			return version{}, fmt.Errorf("invalid Grafana version %q", s)
		}
		n[i] = v
	}
	return version{n[0], n[1], n[2]}, nil
}

func (v version) less(o version) bool {
	if v.major != o.major {
		return v.major < o.major
	}
	if v.minor != o.minor {
		return v.minor < o.minor
	}
	return v.patch < o.patch
}

func (v version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
}

// MinGrafana sets the oldest Grafana version the plugin supports. A check for
// a Grafana behavior that every supported version has fixed then predicts the
// fixed behavior, and reports it as advice. Without MinGrafana, the checks
// assume the plugin supports versions without any fix. MinGrafana panics if v
// is not a version such as "12.3.0".
func MinGrafana(v string) Option {
	parsed, err := parseVersion(v)
	if err != nil {
		panic("sqlexprtest: " + err.Error())
	}
	return func(o *options) { o.minGrafana = &parsed }
}

// MinGrafanaFromPluginJSON reads the oldest Grafana version the plugin
// supports from the grafanaDependency range in its plugin.json, and returns it
// as a [MinGrafana] option.
func MinGrafanaFromPluginJSON(path string) (Option, error) {
	b, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return nil, err
	}
	var p struct {
		Dependencies struct {
			GrafanaDependency string `json:"grafanaDependency"`
		} `json:"dependencies"`
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if p.Dependencies.GrafanaDependency == "" {
		return nil, fmt.Errorf("%s has no dependencies.grafanaDependency", path)
	}
	v, err := lowestVersion(p.Dependencies.GrafanaDependency)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return func(o *options) { o.minGrafana = &v }, nil
}

// operatorSpace matches a range operator and the spaces after it, as in ">= 12.0.0".
var operatorSpace = regexp.MustCompile(`([<>=^~]+)\s+`)

// lowestVersion returns the lowest version a range such as ">=12.3.0" or
// ">=10.4.0 <12 || >=12.1.0" allows. It reads each lower bound and ignores
// upper bounds.
func lowestVersion(r string) (version, error) {
	var lowest *version
	for alt := range strings.SplitSeq(operatorSpace.ReplaceAllString(r, "$1"), "||") {
		for term := range strings.FieldsSeq(alt) {
			if strings.HasPrefix(term, "<") {
				continue
			}
			v, err := parseVersion(strings.TrimLeft(term, ">=^~"))
			if err != nil {
				return version{}, fmt.Errorf("grafanaDependency %q: %w", r, err)
			}
			if lowest == nil || v.less(*lowest) {
				lowest = &v
			}
		}
	}
	if lowest == nil {
		return version{}, fmt.Errorf("grafanaDependency %q has no lower bound", r)
	}
	return *lowest, nil
}

// fixedIn returns the Grafana version that fixes the behavior a check
// reports, and whether every version the plugin supports has the fix.
func (r *Report) fixedIn(code Code) (string, bool) {
	s, ok := grafanaFixes[code]
	if !ok {
		return "", false
	}
	fixed, err := parseVersion(s)
	if err != nil {
		panic("sqlexprtest: grafanaFixes: " + err.Error())
	}
	return fixed.String(), r.minGrafana != nil && !r.minGrafana.less(fixed)
}
