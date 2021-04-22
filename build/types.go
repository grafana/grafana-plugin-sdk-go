package build

// Config holds the setup variables required for a build
type Config struct {
	OS          string // GOOS
	Arch        string // GOOS
	CustomVars  map[string]string
	EnableDebug bool
	Env         map[string]string
	EnableCGo   bool
}

// BeforeBuildCallback hooks into the build process
type BeforeBuildCallback func(cfg Config) (Config, error)
