package build

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

// Callbacks give you a way to run custom behavior when things happen
var beforeBuild = func(cfg Config) (Config, error) {
	return cfg, nil
}

// SetBeforeBuildCallback configures a custom callback
func SetBeforeBuildCallback(cb BeforeBuildCallback) error {
	beforeBuild = cb
	return nil
}

var exname string

func getExecutableName(os string, arch string) (string, error) {
	if exname == "" {
		exename, err := getExecutableFromPluginJSON()
		if err != nil {
			return "", err
		}

		exname = exename
	}

	exeName := fmt.Sprintf("%s_%s_%s", exname, os, arch)
	if os == "windows" {
		exeName = fmt.Sprintf("%s.exe", exeName)
	}
	return exeName, nil
}

func getExecutableFromPluginJSON() (string, error) {
	byteValue, err := ioutil.ReadFile(path.Join("src", "plugin.json"))
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	err = json.Unmarshal(byteValue, &result)
	if err != nil {
		return "", err
	}
	executable := result["executable"]
	name, ok := executable.(string)
	if !ok || name == "" {
		return "", fmt.Errorf("plugin.json is missing an executable name")
	}
	return name, nil
}

func findRunningPIDs(exe string) []int {
	pids := []int{}
	out, err := sh.Output("pgrep", "-f", exe)
	if err != nil || out == "" {
		return pids
	}
	for _, txt := range strings.Fields(out) {
		pid, err := strconv.Atoi(txt)
		if err == nil {
			pids = append(pids, pid)
		} else {
			log.Printf("Unable to format %s (%s)", txt, err)
		}
	}
	return pids
}

func killAllPIDs(pids []int) error {
	for _, pid := range pids {
		log.Printf("Killing process: %d", pid)
		err := syscall.Kill(pid, 9)
		if err != nil {
			return err
		}
	}
	return nil
}

func buildBackend(cfg Config) error {
	cfg, err := beforeBuild(cfg)
	if err != nil {
		return err
	}

	exeName, err := getExecutableName(cfg.OS, cfg.Arch)
	if err != nil {
		return err
	}

	args := []string{
		"build", "-o", path.Join("dist", exeName), "-tags", "netgo",
	}
	if cfg.EnableDebug {
		args = append(args, "-gcflags=all=-N -l")
	} else {
		args = append(args, "-ldflags", "-w")
	}
	args = append(args, "./pkg")

	cfg.Env["GOARCH"] = cfg.Arch
	cfg.Env["GOOS"] = cfg.OS

	// TODO: Change to sh.RunWithV once available.
	return sh.RunWith(cfg.Env, "go", args...)
}

func newBuildConfig(os string, arch string) Config {
	return Config{
		OS:          os,
		Arch:        arch,
		EnableDebug: false,
		Env:         map[string]string{},
	}
}

// Build is a namespace.
type Build mg.Namespace

// Linux builds the back-end plugin for Linux.
func (Build) Linux() error {
	return buildBackend(newBuildConfig("linux", "amd64"))
}

// Windows builds the back-end plugin for Windows.
func (Build) Windows() error {
	return buildBackend(newBuildConfig("windows", "amd64"))
}

// Darwin builds the back-end plugin for OSX.
func (Build) Darwin() error {
	return buildBackend(newBuildConfig("darwin", "amd64"))
}

// Debug builds the debug version for the current platform
func (Build) Debug() error {
	cfg := newBuildConfig(runtime.GOOS, runtime.GOARCH)
	cfg.EnableDebug = true
	return buildBackend(cfg)
}

// Backend build a production build for all platforms
func (Build) Backend() {
	b := Build{}
	mg.Deps(b.Linux, b.Windows, b.Darwin)
}

// BuildAll builds production back-end components.
func BuildAll() { //revive:disable-line
	b := Build{}
	mg.Deps(b.Backend)
}

// Test runs backend tests.
func Test() error {
	if err := sh.RunV("go", "test", "./pkg/..."); err != nil {
		return err
	}
	return nil
}

// Coverage runs backend tests and makes a coverage report.
func Coverage() error {
	// Create a coverage file if it does not already exist
	if err := os.MkdirAll(filepath.Join(".", "coverage"), os.ModePerm); err != nil {
		return err
	}

	if err := sh.RunV("go", "test", "./pkg/...", "-v", "-cover", "-coverprofile=coverage/backend.out"); err != nil {
		return err
	}

	if err := sh.RunV("go", "tool", "cover", "-html=coverage/backend.out", "-o", "coverage/backend.html"); err != nil {
		return err
	}

	return nil
}

// Lint audits the source style
func Lint() error {
	return sh.RunV("golangci-lint", "run", "./...")
}

// Format formats the sources.
func Format() error {
	if err := sh.RunV("gofmt", "-w", "."); err != nil {
		return err
	}

	return nil
}

// Clean cleans build artifacts, by deleting the dist directory.
func Clean() error {
	err := os.RemoveAll("dist")
	if err != nil {
		return err
	}

	err = os.RemoveAll("coverage")
	if err != nil {
		return err
	}

	err = os.RemoveAll("ci")
	if err != nil {
		return err
	}
	return nil
}

// checkLinuxPtraceScope verifies that ptrace is configured as required.
func checkLinuxPtraceScope() error {
	ptracePath := "/proc/sys/kernel/yama/ptrace_scope"
	byteValue, err := ioutil.ReadFile(ptracePath)
	if err != nil {
		return fmt.Errorf("unable to read ptrace_scope: %w", err)
	}
	val := strings.TrimSpace(string(byteValue))
	if val != "0" {
		log.Printf("WARNING:")
		fmt.Printf("ptrace_scope set to value other than 0 (currently: %s), this might prevent debugger from connecting\n", val)
		fmt.Printf("try writing \"0\" to %s\n", ptracePath)
		fmt.Printf("Set ptrace_scope to 0? y/N (default N)\n")

		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			if scanner.Text() == "y" || scanner.Text() == "Y" {
				// if err := sh.RunV("echo", "0", "|", "sudo", "tee", ptracePath); err != nil {
				// 	return // Error?
				// }
				log.Printf("TODO, run: echo 0 | sudo tee /proc/sys/kernel/yama/ptrace_scope")
			} else {
				fmt.Printf("Did not write\n")
			}
		}
	}

	return nil
}

// ReloadPlugin - kills any running instances and waits for grafana to reload the plugin
func ReloadPlugin() error {
	exeName, err := getExecutableName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}

	_ = killAllPIDs(findRunningPIDs(exeName))
	_ = sh.RunV("pkill", "dlv")

	// Wait for grafana to start plugin
	for i := 0; i < 20; i++ {
		time.Sleep(250 * time.Millisecond)
		pids := findRunningPIDs(exeName)
		if len(pids) > 1 {
			log.Printf("multiple instances already running")
			break
		}
		if len(pids) > 0 {
			pid := strconv.Itoa(pids[0])
			log.Printf("Running PID: %s", pid)
			break
		}

		log.Printf("waiting for grafana to start: %s...", exeName)
	}
	return nil
}

// Debugger makes a new debug build, re-launches the plugin and attaches the Delve debugger, in headless mode
// listening on port 3222.
//
// The plugin process is killed after re-building, in order to make Grafana launch the new version. Once the new
// version is up, we attach to it with Delve.
func Debugger() error {
	// Debug build
	b := Build{}
	mg.Deps(b.Debug)

	// 1. kill any running instance
	exeName, err := getExecutableName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	_ = killAllPIDs(findRunningPIDs(exeName))
	_ = sh.RunV("pkill", "dlv")
	if runtime.GOOS == "linux" {
		if err := checkLinuxPtraceScope(); err != nil {
			return err
		}
	}

	// Wait for grafana to start plugin
	pid := -1
	for i := 0; i < 20; i++ {
		pids := findRunningPIDs(exeName)
		if len(pids) > 1 {
			return fmt.Errorf("multiple instances already running")
		}
		if len(pids) > 0 {
			pid = pids[0]
			log.Printf("Found plugin PID: %d", pid)
			break
		}

		log.Printf("Waiting for Grafana to start plugin: %q...", exeName)
		time.Sleep(250 * time.Millisecond)
	}
	if pid == -1 {
		return fmt.Errorf("could not find plugin process %q, perhaps Grafana is not running?", exeName)
	}

	pidStr := strconv.Itoa(pid)
	log.Printf("Attaching Delve to plugin process %d", pid)
	if err := sh.RunV("dlv",
		"attach",
		pidStr,
		"--headless",
		"--listen=:3222",
		"--api-version", "2",
		"--log"); err != nil {
		return err
	}
	log.Printf("Delve finished successfully")

	return nil
}
