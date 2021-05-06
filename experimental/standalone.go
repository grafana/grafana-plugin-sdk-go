package experimental

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
)

type standaloneArgs struct {
	address    string
	standalone bool
	debugger   bool
	dir        string
}

// DoGRPC looks at the environment properties and decides if this should run as a normal hashicorp plugin or
// as a standalone gRPC server
func DoGRPC(id string, opts datasource.ServeOpts) error {
	backend.SetupPluginEnvironment(id) // Enable profiler

	info, err := getStandaloneInfo(id)
	if err != nil {
		return err
	}

	if info.standalone {
		return backend.StandaloneServe(backend.ServeOpts{
			CheckHealthHandler:  opts.CheckHealthHandler,
			CallResourceHandler: opts.CallResourceHandler,
			QueryDataHandler:    opts.QueryDataHandler,
			StreamHandler:       opts.StreamHandler,
			GRPCSettings:        opts.GRPCSettings,
		}, info.address)
	} else if info.address != "" {
		runDummyPluginLocator(info.address)
		return nil
	}

	// The default/normal hashicorp path
	return datasource.Serve(opts)
}

// ManageGRPC ...
func ManageGRPC(id string, factoryFunc datasource.InstanceFactoryFunc, opts datasource.ManageOpts) error {
	backend.SetupPluginEnvironment(id) // Enable profiler

	info, err := getStandaloneInfo(id)
	if err != nil {
		return err
	}

	if info.standalone {
		autoManager := datasource.NewAutoInstanceManager(datasource.NewInstanceManager(factoryFunc))
		return backend.StandaloneServe(backend.ServeOpts{
			CheckHealthHandler:  autoManager,
			CallResourceHandler: autoManager,
			QueryDataHandler:    autoManager,
			StreamHandler:       autoManager,
			GRPCSettings:        opts.GRPCSettings,
		}, info.address)
	} else if info.address != "" {
		runDummyPluginLocator(info.address)
		return nil
	}

	// The default/normal hashicorp path
	return datasource.Manage(factoryFunc, opts)
}

func getStandaloneInfo(id string) (standaloneArgs, error) {
	info := standaloneArgs{}

	var standalone bool
	var address string
	flag.BoolVar(&standalone, "standalone", false, "should this run standalone")
	flag.Parse()

	info.standalone = standalone

	// standalone path
	ex, err := os.Executable()
	if err != nil {
		return info, err
	}

	// When debugging in vscode, write the file in `dist`
	if standalone && strings.HasSuffix(ex, "/pkg/__debug_bin") {
		info.debugger = true
		port, err := getFreePort()
		if err == nil {
			address = fmt.Sprintf(":%d", port)
		}
		ex = filepath.Join(filepath.Dir(ex), "..", "dist", "exe")
	}
	info.dir = filepath.Dir(ex)
	filePath := filepath.Join(info.dir, "standalone.txt")

	// Address from environment variable
	if address == "" {
		envvar := "GF_PLUGIN_GRPC_ADDRESS_" + strings.ReplaceAll(strings.ToUpper(id), "-", "_")
		address = os.Getenv(envvar)
	}

	// Check the local file for address
	addrBytes, err := ioutil.ReadFile(filePath)
	if address == "" {
		if err == nil && len(addrBytes) > 0 {
			address = string(addrBytes)
		}
	}
	info.address = address

	// Write the address to the local file
	if standalone {
		if info.address == "" {
			return info, fmt.Errorf("standalone address must be specified")
		}
		_ = ioutil.WriteFile(filePath, []byte(info.address), 0600)
		// sadly vs-code can not listen to shutdown events
		// https://github.com/golang/vscode-go/issues/120

		// When debugging, be sure to kill the running instances so we reconnect
		if info.debugger {
			findAndKillCurrentPlugin(info.dir)
		}
	}
	return info, nil
}

func runDummyPluginLocator(address string) {
	fmt.Printf("1|2|tcp|%s|grpc\n", address)
	t := time.NewTicker(time.Second * 10)

	for ts := range t.C {
		fmt.Printf("[%s] using address: %s\n", ts.Format("2006-01-02 15:04:05"), address)
	}
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}

	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// Killing the currently registered plugin will cause grafana to restart it
// this time pointing to our new host
func findAndKillCurrentPlugin(dir string) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Error finding processes", r)
		}
	}()

	var pluginJSON map[string]interface{}
	pjson, err := ioutil.ReadFile(filepath.Join(dir, "plugin.json"))
	if err != nil {
		return
	}
	err = json.Unmarshal(pjson, &pluginJSON)
	if err != nil {
		return
	}
	exeprefix, ok := pluginJSON["executable"]
	if !ok {
		fmt.Printf("missing executable form plugin.json")
		return
	}

	arg1 := exeprefix.(string)
	out, err := exec.Command("pgrep", arg1).Output()
	if err != nil {
		fmt.Printf("error running pgrep")
		return
	}
	for _, txt := range strings.Fields(string(out)) {
		pid, err := strconv.Atoi(txt)
		if err == nil {
			log.Printf("Killing process: %d", pid)
			// err := syscall.Kill(pid, 9)
			pidstr := fmt.Sprintf("%d", pid)
			err = exec.Command("kill", "-9", pidstr).Run()
			if err != nil {
				log.Printf("Error: %s", err.Error())
			}
		}
	}
}
