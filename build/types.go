package build

// BuildConfig holds the setup variables required for a build
type BuildConfig struct { //revive:disable-line
	OS          string // GOOS
	Arch        string // GOOS
	EnableDebug bool
	Env         map[string]string
}
