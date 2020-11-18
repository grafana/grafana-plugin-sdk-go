package build

type ArtifactInfo struct {
	Path     string `json:"name,omitempty"`
	Size     int64  `json:"size,omitempty"`
	Sha1     string `json:"sha1,omitempty"`
	MD5      string `json:"md5,omitempty"`
	Platform string `json:"platform,omitempty"`
}

type GitCommitInfo struct {
	Commit    string `json:"commit,omitempty"`
	Subject   string `json:"subject,omitempty"`
	Author    string `json:"author,omitempty"`
	Committer string `json:"commiter,omitempty"`
}

// See:
// https://storage.googleapis.com/plugins-ci/ci/grafana-timestream-datasource/branch/update-auth/281/report.json

// Config holds the setup variables required for a build
type Report struct {
	UID         string `json:"uid,omitempty"`
	BuildNum    int64  `json:"buildNum,omitempty"`
	Repo        string `json:"repo,omitempty"` // grafana/timestream-datasource
	Status      string `json:"status,omitempty"`
	StartTime   int64  `json:"startTime,omitempty"`
	EndTime     int64  `json:"endTime,omitempty"` // will really be the time we write the report!
	Branch      string `json:"branch,omitempty"`  // master, v3.4.x, etc
	PullRequest int64  `json:"pr,omitempty"`      // 0 or number
	Path        string `json:"path,omitempty"`    // Path relative to the root

	// Searchable values
	Category string   `json:"category,omitempty"` // "plugin" | "grafana" | "metrics"
	Product  string   `json:"product,omitempty"`  // plugin id
	Version  string   `json:"version,omitempty"`  // plugin version
	Tags     []string `json:"tags,omitempty"`     // searchable strings "enterprise", "edge", etc

	// List of useable artifacts
	Artifacts []ArtifactInfo `json:"artifacts,omitempty"`

	// Information about the git commit
	GitInfo GitCommitInfo `json:"git,omitempty"`

	// Arbitrary metrics collected for this build.
	Metrics map[string]float64
}

// PUNT For now:
// * coverage reports
// * test results (junit browser)
// * cypress results (images etc)

// Queries we want to make sure are easy in SQL
//-----------------------------------------------
// Give me the latest master linux artifact for iot-sitewise
// Give me the latest master linux artifacts for all enterprise plugins
// Give me the latest master linux artifacts for all "edge" plugins
//
