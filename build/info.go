package build

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// exposed for testing.
var now = time.Now

// Info See also PluginBuildInfo in https://github.com/grafana/grafana/blob/master/pkg/plugins/models.go
type Info struct {
	Time   int64  `json:"time,omitempty"`
	Repo   string `json:"repo,omitempty"`
	Branch string `json:"branch,omitempty"`
	Hash   string `json:"hash,omitempty"`
	Build  int64  `json:"build,omitempty"`
	PR     int64  `json:"pr,omitempty"`
}

func (v Info) appendFlags(flags map[string]string, prefix string) {
	if v.Repo != "" {
		flags[prefix+"repo"] = v.Repo
	}
	if v.Branch != "" {
		flags[prefix+"branch"] = v.Branch
	}
	if v.Hash != "" {
		flags[prefix+"hash"] = v.Hash
	}
	if v.Build > 0 {
		flags[prefix+"build"] = fmt.Sprintf("%d", v.Build)
	}
	if v.PR > 0 {
		flags[prefix+"PR"] = fmt.Sprintf("%d", v.PR)
	}
}

func getEnvironment(check ...string) string {
	for _, key := range check {
		if strings.HasPrefix(key, "> ") {
			parts := strings.Split(key, " ")
			cmd := exec.Command(parts[1], parts[2:]...) // #nosec G204
			out, err := cmd.CombinedOutput()
			if err == nil && len(out) > 0 {
				str := strings.TrimSpace(string(out))
				if strings.Index(str, " ") > 0 {
					continue // skip any output that has spaces
				}
				return str
			}
			continue
		}

		val := os.Getenv(key)
		if val != "" {
			return strings.TrimSpace(val)
		}
	}
	return ""
}

// GetBuildInfoFromEnvironment reads the
func GetBuildInfoFromEnvironment() Info {
	v := Info{
		Time: now().UnixNano() / int64(time.Millisecond),
	}

	v.Repo = getEnvironment("DRONE_REPO_LINK", "CIRCLE_PROJECT_REPONAME", "> git remote get-url origin")
	v.Branch = getEnvironment("DRONE_BRANCH", "CIRCLE_BRANCH", "> git branch --show-current")
	v.Hash = getEnvironment("DRONE_COMMIT_SHA", "CIRCLE_SHA1", "> git rev-parse HEAD")
	val, err := strconv.ParseInt(getEnvironment("DRONE_BUILD_NUMBER", "CIRCLE_BUILD_NUM"), 10, 64)
	if err == nil {
		v.Build = val
	}
	val, err = strconv.ParseInt(getEnvironment("DRONE_PULL_REQUEST", "CI_PULL_REQUEST"), 10, 64)
	if err == nil {
		v.PR = val
	}
	return v
}
