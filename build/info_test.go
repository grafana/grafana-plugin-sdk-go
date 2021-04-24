package build

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFillBuildInfo(t *testing.T) {
	// Set this as a constant for testing
	now = func() time.Time { return time.Unix(1515151515, 0) }
	t.Cleanup(func() {
		now = time.Now
	})

	t.Run("drone", func(t *testing.T) {
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
		})

		info := getBuildInfoFromEnvironment()
		require.NotNil(t, info)
		assert.Equal(t, "main", info.Branch)
		assert.Equal(t, "bcdd4bf0245c82c060407b3b24b9b87301d15ac1", info.Hash)
		assert.Equal(t, int64(22), info.Build)
		assert.Equal(t, int64(33), info.PR)
	})

	t.Run("circle", func(t *testing.T) {
		os.Setenv("CIRCLE_PROJECT_REPONAME", "https://github.com/octocat/hello-world")
		os.Setenv("CIRCLE_BRANCH", "main")
		os.Setenv("CIRCLE_SHA1", "bcdd4bf0245c82c060407b3b24b9b87301d15ac1")
		os.Setenv("CIRCLE_BUILD_NUM", "22")
		os.Setenv("CI_PULL_REQUEST", "33")
		t.Cleanup(func() {
			_ = os.Unsetenv("CIRCLE_PROJECT_REPONAME")
			_ = os.Unsetenv("CIRCLE_BRANCH")
			_ = os.Unsetenv("CIRCLE_SHA1")
			_ = os.Unsetenv("CIRCLE_BUILD_NUM")
			_ = os.Unsetenv("CI_PULL_REQUEST")
		})

		info := getBuildInfoFromEnvironment()
		require.NotNil(t, info)
		assert.Equal(t, "main", info.Branch)
		assert.Equal(t, "bcdd4bf0245c82c060407b3b24b9b87301d15ac1", info.Hash)
		assert.Equal(t, int64(22), info.Build)
		assert.Equal(t, int64(33), info.PR)
	})

	// really testable since it delegates to functions, but helful in local dev
	t.Run("git commands", func(t *testing.T) {
		info := getBuildInfoFromEnvironment()
		fmt.Printf("BUILD: %#v\n", info)
		require.NotNil(t, info)
	})
}
