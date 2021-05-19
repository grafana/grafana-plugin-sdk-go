package plugintest

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/onsi/gomega/gexec"
)

// ShutdownFunc is meant to be called to clean up resources created and in use when a plugin is started
type ShutdownFunc func()

// StartPlugin compiles and starts the backend datasource plugin at packagePath.
// It listens on the port and passes the provided env to the plugin.
// id should be the same as the plugin id defined in package.json
func StartPlugin(packagePath string, id string, port int, env ...string) (*PluginClient, ShutdownFunc, error) {
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	env = setupEnv(id, addr, env)

	execPath, err := gexec.Build(packagePath)
	if err != nil {
		gexec.CleanupBuildArtifacts()
		return nil, func() {}, err
	}

	shutdownPlugin, err := startPlugin(execPath, env)
	if err != nil {
		shutdownPlugin()
		gexec.CleanupBuildArtifacts()
		return nil, func() {}, err
	}

	plugin, disconnect, err := NewPluginClient(addr)
	if err != nil {
		disconnect()
		shutdownPlugin()
		gexec.CleanupBuildArtifacts()
		return nil, func() {}, err
	}

	return plugin, func() {
		disconnect()
		shutdownPlugin()
		gexec.CleanupBuildArtifacts()
	}, nil
}

func setupEnv(id string, addr string, env []string) []string {
	addrEnv := "GF_PLUGIN_GRPC_ADDRESS_" + strings.ReplaceAll(strings.ToUpper(id), "-", "_")
	return append(env, fmt.Sprintf("%s=%s", addrEnv, addr))
}

func startPlugin(execPath string, env []string) (ShutdownFunc, error) {
	cmd := exec.Command(execPath, "--standalone=true")
	cmd.Env = env

	process, err := gexec.Start(cmd, os.Stdout, os.Stderr)
	if err != nil {
		return func() {}, err
	}

	shutdown := func() {
		process.Terminate()
	}

	return shutdown, err
}
