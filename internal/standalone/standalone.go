package standalone

import (
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

	"github.com/grafana/grafana-plugin-sdk-go/internal"
)

type Args struct {
	Address    string
	Standalone bool
	debugger   bool
	dir        string
}

func GetInfo(id string) (Args, error) {
	info := Args{}

	var standalone bool
	var address string
	flag.BoolVar(&standalone, "standalone", false, "should this run standalone")
	flag.Parse()

	info.Standalone = standalone

	// standalone path
	ex, err := os.Executable()
	if err != nil {
		return info, err
	}

	// When debugging in vscode, write the file in `dist`
	if standalone && strings.HasPrefix(filepath.Base(ex), "__debug_bin") {
		info.debugger = true
		port, err := getFreePort()
		if err == nil {
			address = fmt.Sprintf(":%d", port)
		}
		js, err := findPluginJSON(ex)
		if err != nil {
			return info, err
		}
		ex = js
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
	info.Address = address

	// Write the address to the local file
	if standalone {
		if info.Address == "" {
			return info, fmt.Errorf("standalone address must be specified")
		}
		_ = ioutil.WriteFile(filePath, []byte(info.Address), 0600)
		// sadly vs-code can not listen to shutdown events
		// https://github.com/golang/vscode-go/issues/120

		// When debugging, be sure to kill the running instances so we reconnect
		if info.debugger {
			findAndKillCurrentPlugin(info.dir)
		}
	}
	return info, nil
}

// will check a few options to find the dist plugin json file
func findPluginJSON(exe string) (string, error) {
	cwd, _ := os.Getwd()
	if filepath.Base(cwd) == "pkg" {
		cwd = filepath.Join(cwd, "..")
	}

	check := []string{
		filepath.Join(filepath.Dir(exe), "plugin.json"),
		filepath.Join(filepath.Dir(exe), "..", "dist", "plugin.json"),
		filepath.Join(cwd, "dist", "plugin.json"),
		filepath.Join(cwd, "plugin.json"),
	}

	for _, path := range check {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return exe, fmt.Errorf("can not find plugin.json in: %v", check)
}

func RunDummyPluginLocator(address string) {
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
	defer func() { _ = l.Close() }()
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

	exeprefix, err := internal.GetExecutableFromPluginJSON(dir)
	if err != nil {
		fmt.Printf("missing executable in plugin.json (standalone)")
		return
	}

	out, err := exec.Command("pgrep", exeprefix).Output()
	if err != nil {
		fmt.Printf("error running pgrep: %s (%s)", err.Error(), exeprefix)
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
