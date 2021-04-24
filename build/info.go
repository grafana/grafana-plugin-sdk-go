package build

import (
	"os"
	"strconv"
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

func getEnvironment(check ...string) string {
	for _, key := range check {
		val := os.Getenv(key)
		if val != "" {
			return val
		}
	}
	return ""
}

// GetBuildInfoFromEnvironment reads the
func GetBuildInfoFromEnvironment() Info {
	v := Info{
		Time: now().UnixNano() / int64(time.Millisecond),
	}

	v.Repo = getEnvironment("DRONE_REPO_LINK", "CIRCLE_PROJECT_REPONAME")
	v.Branch = getEnvironment("DRONE_BRANCH", "CIRCLE_BRANCH")
	v.Hash = getEnvironment("DRONE_COMMIT_SHA", "CIRCLE_SHA1")
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
