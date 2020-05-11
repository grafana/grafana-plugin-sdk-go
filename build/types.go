package build

// BuildConfig holds the setup variables required for a build
type BuildConfig struct {
	OS          string // GOOS
	ARCH        string // GOOS
	enableDebug bool
	env         map[string]string
}
