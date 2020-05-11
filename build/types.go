package build

// BuildConfig holds the setup variables required for a build
type BuildConfig struct {
	OS          string // GOOS
	Arch        string // GOOS
	EnableDebug bool
	Env         map[string]string
}
