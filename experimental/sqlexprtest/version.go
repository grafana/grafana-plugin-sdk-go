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
// the behavior it reports. Every later release must also have the fix: a fix
// that is backported to some release lines but not others can't be listed
// here yet. FixedIn names the fix release. It doesn't mean the plugin's
// minimum version has the fix.
var grafanaFixes = map[Code]string{}

type version struct {
	major, minor, patch int
	prerelease          string
}

var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)(?:\.(0|[1-9][0-9]*))?(?:\.(0|[1-9][0-9]*))?(?:-([0-9A-Za-z.-]+))?(?:\+([0-9A-Za-z.-]+))?$`)

// parseVersion reads a version such as "13.3.0", "v13.3.0-pre", or "13". A
// missing minor or patch is 0. Build metadata after "+" is ignored.
func parseVersion(s string) (version, error) {
	bad := fmt.Errorf("invalid Grafana version %q", s)
	m := versionPattern.FindStringSubmatch(strings.TrimPrefix(strings.TrimSpace(s), "v"))
	// A prerelease or build suffix needs all three numbers, as in 13.3.0-pre.
	if m == nil || m[3] == "" && (m[4] != "" || m[5] != "") {
		return version{}, bad
	}
	var n [3]int
	for i := range n {
		if m[i+1] == "" {
			break
		}
		var err error
		if n[i], err = strconv.Atoi(m[i+1]); err != nil {
			return version{}, bad
		}
	}
	for _, suffix := range m[4:6] {
		if suffix != "" && strings.Contains("."+suffix+".", "..") {
			return version{}, bad
		}
	}
	return version{n[0], n[1], n[2], m[4]}, nil
}

func numericIdentifier(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}

// less orders versions as semver does: 13.3.0-pre comes before 13.3.0.
func (v version) less(o version) bool {
	if v.major != o.major {
		return v.major < o.major
	}
	if v.minor != o.minor {
		return v.minor < o.minor
	}
	if v.patch != o.patch {
		return v.patch < o.patch
	}
	if v.prerelease == "" || o.prerelease == "" {
		return v.prerelease != "" && o.prerelease == ""
	}
	a, b := strings.Split(v.prerelease, "."), strings.Split(o.prerelease, ".")
	for i := range min(len(a), len(b)) {
		if a[i] == b[i] {
			continue
		}
		an, bn := numericIdentifier(a[i]), numericIdentifier(b[i])
		if an != bn {
			return an
		}
		if an && len(a[i]) != len(b[i]) {
			return len(a[i]) < len(b[i])
		}
		return a[i] < b[i]
	}
	return len(a) < len(b)
}

func (v version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch)
	if v.prerelease != "" {
		s += "-" + v.prerelease
	}
	return s
}

// MinGrafana sets the oldest Grafana version the plugin supports. A
// prerelease comes before its release, so a minimum of 13.3.0-pre does not
// turn on a fix released in 13.3.0. Without MinGrafana, the checks assume the
// plugin supports versions without any fix. MinGrafana panics if v is not a
// version such as "12.3.0".
func MinGrafana(v string) Option {
	parsed, err := parseVersion(v)
	if err != nil {
		panic("sqlexprtest: " + err.Error())
	}
	return func(o *options) { o.minGrafana = &parsed }
}

// MinGrafanaFromPluginJSON reads the oldest Grafana version that the
// grafanaDependency range in plugin.json allows, and returns it as a
// [MinGrafana] option. It reads ranges such as ">=12.3.0", "^11.0.0",
// "~12.1.0", ">=10.4.0 <12 || >=12.1.0", and a plain version. Other range
// syntax, such as "12.x" or ">12.0.0", returns an error. Then pass the
// version to MinGrafana yourself.
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

// lowestVersion returns the lowest version a grafanaDependency range allows.
// In each "||" alternative, the lowest version is the highest lower bound
// (>=, ^, ~, =, or a plain version), or 0.0.0 when there is none. Upper
// bounds (<, <=) only check that the alternative allows some version.
func lowestVersion(r string) (version, error) {
	var lowest *version
	for alt := range strings.SplitSeq(operatorSpace.ReplaceAllString(r, "$1"), "||") {
		terms := strings.Fields(alt)
		if len(terms) == 0 {
			return version{}, fmt.Errorf("grafanaDependency %q has an empty alternative", r)
		}
		var low version
		var uppers []string
		for _, term := range terms {
			op, rest := splitOperator(term)
			v, err := parseVersion(rest)
			if err != nil {
				return version{}, fmt.Errorf("grafanaDependency %q: %w", r, err)
			}
			switch op {
			case ">=", "^", "~", "=", "":
				if low.less(v) {
					low = v
				}
			case "<", "<=":
				uppers = append(uppers, term)
			default:
				return version{}, fmt.Errorf("grafanaDependency %q: unsupported operator %q; use MinGrafana", r, op)
			}
		}
		for _, term := range uppers {
			op, rest := splitOperator(term)
			upper, _ := parseVersion(rest)
			if op == "<" && !low.less(upper) || op == "<=" && upper.less(low) {
				return version{}, fmt.Errorf("grafanaDependency %q: %q allows no versions", r, alt)
			}
		}
		if lowest == nil || low.less(*lowest) {
			lowest = &low
		}
	}
	return *lowest, nil
}

func splitOperator(term string) (string, string) {
	i := strings.IndexFunc(term, func(c rune) bool { return !strings.ContainsRune("<>=^~!", c) })
	if i < 0 {
		return term, ""
	}
	return term[:i], term[i:]
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
