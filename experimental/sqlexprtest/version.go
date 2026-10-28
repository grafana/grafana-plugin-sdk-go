package sqlexprtest

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// grafanaFixes contains monotonic fixes: every release at or after the named
// version must contain the fix. Backports with unfixed later releases need
// release-range rules before they can be represented here. FixedIn names a
// known fix; it does not mean the selected minimum contains that fix.
var grafanaFixes = map[Code]string{
	CodeEnumField:          "13.3.0",
	CodeGeneratedNameClash: "13.3.0",
}

type version struct {
	major, minor, patch int
	prerelease          string
}

var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)(?:\.(0|[1-9][0-9]*))?(?:\.(0|[1-9][0-9]*))?(?:-([0-9A-Za-z.-]+))?(?:\+([0-9A-Za-z.-]+))?$`)

func parseVersion(s string) (version, error) {
	v, _, err := parseVersionParts(s)
	return v, err
}

func parseVersionParts(s string) (version, int, error) {
	bad := func() error {
		return fmt.Errorf("invalid Grafana version %q", s)
	}
	m := versionPattern.FindStringSubmatch(strings.TrimPrefix(strings.TrimSpace(s), "v"))
	if m == nil || m[3] == "" && (m[4] != "" || m[5] != "") {
		return version{}, 0, bad()
	}
	var n [3]int
	parts := 0
	for i := range n {
		if m[i+1] == "" {
			break
		}
		var err error
		n[i], err = strconv.Atoi(m[i+1])
		if err != nil {
			return version{}, 0, bad()
		}
		parts++
	}
	for i, suffix := range m[4:6] {
		if suffix == "" {
			continue
		}
		for id := range strings.SplitSeq(suffix, ".") {
			if id == "" || i == 0 && numericIdentifier(id) && len(id) > 1 && id[0] == '0' {
				return version{}, 0, bad()
			}
		}
	}
	return version{n[0], n[1], n[2], m[4]}, parts, nil
}

func numericIdentifier(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return s != ""
}

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

func (v version) release() version {
	v.prerelease = ""
	return v
}

// MinGrafana sets the oldest Grafana version the plugin supports. Prereleases
// precede the final release, so a minimum of 13.3.0-pre does not enable a fix in
// 13.3.0. Build metadata does not affect precedence. Major-only and major.minor
// versions are shorthand for zero minor/patch components.
//
// Without MinGrafana, checks assume versions without any fix are supported.
// MinGrafana panics if v is not a valid version.
func MinGrafana(v string) Option {
	parsed, err := parseVersion(v)
	if err != nil {
		panic("sqlexprtest: " + err.Error())
	}
	return func(o *options) { o.minGrafana = &parsed }
}

// MinGrafanaFromPluginJSON reads the minimum allowed by grafanaDependency.
// It supports exact/partial versions, < <= > >= = comparators, caret and tilde
// ranges, whitespace intersections, and || alternatives. Prereleases require
// an explicit prerelease comparator with the same major.minor.patch in their
// alternative. Wildcards, hyphen ranges, and other syntax return an error;
// callers can use MinGrafana after establishing the minimum themselves.
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

var operatorSpace = regexp.MustCompile(`([<>=^~]+)\s+`)

type versionConstraint struct {
	op string
	v  version
}

func (c versionConstraint) allows(v version) bool {
	switch c.op {
	case ">=":
		return !v.less(c.v)
	case ">":
		return c.v.less(v)
	case "<=":
		return !c.v.less(v)
	case "<":
		return v.less(c.v)
	default:
		return v == c.v
	}
}

// nextComponent constructs the exclusive upper boundary for a partial,
// caret, or tilde range. The -0 excludes prereleases of that next release too.
func nextComponent(v version, component int) (version, error) {
	n := [3]int{v.major, v.minor, v.patch}
	if n[component] == int(^uint(0)>>1) {
		return version{}, fmt.Errorf("version component too large in %q", v.String())
	}
	n[component]++
	for i := component + 1; i < len(n); i++ {
		n[i] = 0
	}
	return version{n[0], n[1], n[2], "0"}, nil
}

func parseConstraint(term string) ([]versionConstraint, version, error) {
	op := ""
	for _, prefix := range []string{">=", "<=", ">", "<", "=", "^", "~"} {
		if rest, ok := strings.CutPrefix(term, prefix); ok {
			op, term = prefix, rest
			break
		}
	}
	v, parts, err := parseVersionParts(term)
	if err != nil {
		return nil, version{}, err
	}
	original := v
	bound := func(component int) ([]versionConstraint, version, error) {
		upper, err := nextComponent(v, component)
		return []versionConstraint{{">=", v}, {"<", upper}}, original, err
	}
	switch op {
	case "", "=":
		if parts < 3 {
			return bound(parts - 1)
		}
		op = "="
	case "~":
		return bound(min(parts-1, 1))
	case "^":
		component := 0
		if v.major == 0 && parts > 1 {
			component = 1
			if v.minor == 0 && parts > 2 {
				component = 2
			}
		}
		return bound(component)
	case ">", "<=":
		if parts < 3 {
			upper, err := nextComponent(v, parts-1)
			if err != nil {
				return nil, version{}, err
			}
			if op == ">" {
				op, v = ">=", upper.release()
			} else {
				op, v = "<", upper
			}
		}
	case "<":
		if parts < 3 {
			v.prerelease = "0"
		}
	}
	return []versionConstraint{{op, v}}, original, nil
}

func lowestVersion(r string) (version, error) {
	var lowest *version
	for alt := range strings.SplitSeq(operatorSpace.ReplaceAllString(r, "$1"), "||") {
		terms := strings.Fields(alt)
		if len(terms) == 0 {
			return version{}, fmt.Errorf("grafanaDependency %q has an empty alternative", r)
		}
		var constraints []versionConstraint
		prereleases := map[version]bool{}
		for _, term := range terms {
			parsed, original, err := parseConstraint(term)
			if err != nil {
				return version{}, fmt.Errorf("grafanaDependency %q: %w", r, err)
			}
			constraints = append(constraints, parsed...)
			if original.prerelease != "" {
				prereleases[original.release()] = true
			}
		}

		// A minimum is either a stable lower boundary, or a prerelease at an
		// explicitly admitted release. Upper-only alternatives include 0.0.0.
		candidates := []version{{}}
		for release := range prereleases {
			release.prerelease = "0"
			candidates = append(candidates, release)
		}
		for _, c := range constraints {
			candidates = append(candidates, c.v.release())
			if c.v.prerelease != "" {
				candidate := c.v
				if c.op == ">" {
					candidate.prerelease += ".0"
				}
				candidates = append(candidates, candidate)
			} else if c.op == ">" {
				next, err := nextComponent(c.v, 2)
				if err != nil {
					return version{}, fmt.Errorf("grafanaDependency %q: %w", r, err)
				}
				candidates = append(candidates, next.release())
			}
		}
		for _, candidate := range candidates {
			if candidate.prerelease != "" && !prereleases[candidate.release()] {
				continue
			}
			allowed := true
			for _, c := range constraints {
				if !c.allows(candidate) {
					allowed = false
					break
				}
			}
			if allowed && (lowest == nil || candidate.less(*lowest)) {
				v := candidate
				lowest = &v
			}
		}
	}
	if lowest == nil {
		return version{}, fmt.Errorf("grafanaDependency %q allows no versions", r)
	}
	return *lowest, nil
}

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
