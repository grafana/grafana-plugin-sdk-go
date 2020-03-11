//+build mage

package build

import (
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

	"github.com/google/gops/goprocess"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

var exname string = ""

func getExecutableName(os string, arch string) string {
	if exname == "" {
		var err error
		exname, err = getExecutableFromPluginJSON()
		if err != nil {
			exname = "set_exe_name_in_plugin_json" // warning in the final name?
		}
	}

	exeName := fmt.Sprintf("%s_%s_%s", exname, os, arch)
	if "windows" == os {
		exeName = fmt.Sprintf("%s.exe", exeName)
	}
	return exeName
}

func getExecutableFromPluginJSON() (string, error) {
	jsonFile, err := os.Open("src/plugin.json")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = jsonFile.Close()
	}()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	err = json.Unmarshal([]byte(byteValue), &result)
	if err != nil {
		return "", err
	}
	return result["executable"].(string), nil
}

func findRunningProcess(exe string) *goprocess.P {
	for _, process := range goprocess.FindAll() {
		if strings.HasSuffix(process.Path, exe) {
			return &process
		}
	}
	return nil
}

func killProcess(process *goprocess.P) error {
	log.Printf("Killing: %s (%d)", process.Path, process.PID)
	return syscall.Kill(process.PID, 9)
}

func buildBackend(os string, arch string, enableDebug bool) error {
	exeName := getExecutableName(os, arch)

	args := []string{
		"build", "-o", path.Join("dist", exeName), "-tags", "netgo",
	}
	if enableDebug {
		args = append(args, "-gcflags=all=-N -l")
	} else {
		args = append(args, "-ldflags", "-w")
	}
	args = append(args, "./pkg")

	env := map[string]string{
		"GOARCH": arch,
		"GOOS":   os,
	}

	// TODO: Change to sh.RunWithV once available.
	return sh.RunWith(env, "go", args...)
}

// Build is a namespace.
type Build mg.Namespace

// Linux builds the back-end plugin for Linux.
func (Build) Linux() error {
	return buildBackend("linux", "amd64", false)
}

// Windows builds the back-end plugin for Windows.
func (Build) Windows() error {
	return buildBackend("windows", "amd64", false)
}

// Darwin builds the back-end plugin for OSX.
func (Build) Darwin() error {
	return buildBackend("darwin", "amd64", false)
}

// Debug builds the debug version for the current platform
func (Build) Debug() error {
	return buildBackend(runtime.GOOS, runtime.GOARCH, true)
}

// Backend build a production build for all platforms
func (Build) Backend() {
	b := Build{}
	mg.Deps(b.Linux, b.Windows, b.Darwin)
}

// BuildAll builds production back-end components.
func BuildAll() {
	b := Build{}
	mg.Deps(b.Backend)
}

// Test runs backend tests.
func Test() error {
	if err := sh.RunV("go", "test", "./pkg/..."); err != nil {
		return nil
	}
	return nil
}

// Coverage runs backend tests and makes a coverage report.
func Coverage() error {
	// Create a coverage file if it does not already exist
	_ = os.MkdirAll(filepath.Join(".", "coverage"), os.ModePerm)

	if err := sh.RunV("go", "test", "./pkg/...", "-v", "-cover", "-coverprofile=coverage/backend.out"); err != nil {
		return nil
	}

	if err := sh.RunV("go", "tool", "cover", "-html=coverage/backend.out", "-o", "coverage/backend.html"); err != nil {
		return nil
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

// Debugger makes a new debug build and attaches dlv (go-delve).
func Debugger() error {
	// 1. kill any running instance
	exeName := getExecutableName(runtime.GOOS, runtime.GOARCH)

	// Kill any running processs
	process := findRunningProcess(exeName)
	if process != nil {
		err := killProcess(process)
		if err != nil {
			return err
		}
	}

	// Debug build
	b := Build{}
	mg.Deps(b.Debug)

	if runtime.GOOS == "linux" {
		log.Printf("On linux we should check ptrace_scope")
		// 	ptrace_scope=`cat /proc/sys/kernel/yama/ptrace_scope`
		// 	if [ "$ptrace_scope" != 0 ]; then
		// 	  echo "WARNING: ptrace_scope set to value other than 0, this might prevent debugger from connecting, try writing \"0\" to /proc/sys/kernel/yama/ptrace_scope.
		//   Read more at https://www.kernel.org/doc/Documentation/security/Yama.txt"
		// 	  read -p "Set ptrace_scope to 0? y/N (default N)" set_ptrace_input
		// 	  if [ "$set_ptrace_input" == "y" ] || [ "$set_ptrace_input" == "Y" ]; then
		// 		echo 0 | sudo tee /proc/sys/kernel/yama/ptrace_scope
		// 	  fi
		// 	fi
	}

	// Wait for grafana to start plugin
	for i := 0; i < 20; i++ {
		process := findRunningProcess(exeName)
		if process != nil {
			log.Printf("Running PID: %d", process.PID)

			// dlv attach ${PLUGIN_PID} --headless --listen=:${PORT} --api-version 2 --log
			if err := sh.RunV("dlv",
				"attach",
				strconv.Itoa(process.PID),
				"--headless",
				"--listen=:3222",
				"--api-version", "2",
				"--log"); err != nil {
				return err
			}
			// And then kill dvl
			return sh.RunV("pkill", "dlv")
		}

		log.Printf("waiting for grafana to start: %s...", exeName)
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("could not find process: %s, perhaps grafana is not running?", exeName)
}

// Default configures the default target.
var Default = BuildAll
