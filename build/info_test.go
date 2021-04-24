package build

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFillBuildInfo(t *testing.T) {
	// Set this as a constant for testing
	now = func() time.Time { return time.Unix(1515151515, 0) }

	os.Setenv("DRONE_REPO_LINK", "https://github.com/octocat/hello-world")
	os.Setenv("DRONE_BRANCH", "main")
	os.Setenv("DRONE_COMMIT_SHA", "bcdd4bf0245c82c060407b3b24b9b87301d15ac1")
	os.Setenv("DRONE_BUILD_NUMBER", "22")
	os.Setenv("DRONE_PULL_REQUEST", "33")
	t.Cleanup(func() {
		_ = os.Unsetenv("DRONE_REPO_LINK")
		_ = os.Unsetenv("DRONE_BRANCH")
		_ = os.Unsetenv("DRONE_COMMIT_SHA")
		_ = os.Unsetenv("DRONE_BUILD_NUMBER")
		_ = os.Unsetenv("DRONE_PULL_REQUEST")
		now = time.Now
	})

	info := GetBuildInfoFromEnvironment()

	assert.Equal(t, "main", info.Branch)
	assert.Equal(t, "bcdd4bf0245c82c060407b3b24b9b87301d15ac1", info.Hash)
	assert.Equal(t, int64(22), info.Build)
	assert.Equal(t, int64(33), info.PR)
}
