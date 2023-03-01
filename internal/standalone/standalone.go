package standalone

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/internal"
)

type Args struct {
	Address    string
	PID        int
	Standalone bool
	Dir        string
	Debugger   bool
}

// StandaloneAddressFilePath returns the path to the standalone.txt file, which contains the standalone GRPC address
func (a Args) StandaloneAddressFilePath() string {
	return filepath.Join(a.Dir, "standalone.txt")
}

// StandalonePIDFilePath returns the path to the pid.txt file, which contains the standalone GRPC's server PID
func (a Args) StandalonePIDFilePath() string {
	return filepath.Join(a.Dir, "pid.txt")
}

func GetInfo(id string) (Args, error) {
	info := Args{}

	var standalone bool
	var debug bool
	flag.BoolVar(&standalone, "standalone", false, "should this run standalone")
	flag.BoolVar(&debug, "debug", false, "run in debug mode")
	flag.Parse()

	info.Standalone = standalone

	// standalone path
	ex, err := os.Executable()
	if err != nil {
		return info, err
	}

	// VsCode names the file "__debug_bin"
	vsCodeDebug := strings.HasPrefix(filepath.Base(ex), "__debug_bin")
	// GoLand places it in:
	//  Linux: /tmp/GoLand/___XXgo_build_github_com_PACKAGENAME_pkg
	//  Mac OS X: /private/var/folders/lx/XXX/T/GoLand/___go_build_github_com_PACKAGENAME_pkg
	//  Windows: C:\Users\USER\AppData\Local\Temp\GoLand\___go_build_github_com_PACKAGENAME_pkg.exe
	goLandDebug := strings.Contains(ex, "GoLand") && strings.Contains(ex, "go_build_")
	if standalone && (vsCodeDebug || goLandDebug || debug) {
		info.Debugger = true
		js, err := findPluginJSON(ex)
		if err != nil {
			return info, err
		}
		ex = js
	}
	info.Dir = filepath.Dir(ex)

	// Determine standalone address + PID
	info.Address, err = getStandaloneAddress(id, info)
	if err != nil {
		return info, err
	}
	info.PID, err = getStandalonePID(info)
	if err != nil {
		return info, err
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

func getStandaloneAddress(pluginID string, info Args) (string, error) {
	if info.Debugger {
		port, err := getFreePort()
		if err != nil {
			return "", fmt.Errorf("get free port: %w", err)
		}
		return fmt.Sprintf(":%d", port), nil
	}

	// Address from environment variable
	envvar := "GF_PLUGIN_GRPC_ADDRESS_" + strings.ReplaceAll(strings.ToUpper(pluginID), "-", "_")
	if v, ok := os.LookupEnv(envvar); ok {
		return v, nil
	}

	// Check the local file for address
	fb, err := os.ReadFile(info.StandaloneAddressFilePath())
	addressFileContent := string(bytes.TrimSpace(fb))
	switch {
	case err != nil && !os.IsNotExist(err):
		return "", fmt.Errorf("read standalone file: %w", err)
	case os.IsNotExist(err) || len(addressFileContent) == 0:
		// No standalone file, do not treat as standalone
		return "", nil
	}
	return addressFileContent, nil
}

func getStandalonePID(info Args) (int, error) {
	// Read PID (optional, as it was introduced later on)
	fb, err := os.ReadFile(info.StandalonePIDFilePath())
	pidFileContent := string(bytes.TrimSpace(fb))
	switch {
	case err != nil && !os.IsNotExist(err):
		return 0, fmt.Errorf("read pid file: %w", err)
	case os.IsNotExist(err) || len(pidFileContent) == 0:
		// No PID, this is optional as it was introduced later, so it's fine.
		// We lose hot switching between debug and non-debug without the pid file,
		// but there's nothing better we can do.
		return 0, nil
	default:
		pid, err := strconv.Atoi(pidFileContent)
		if err != nil {
			return 0, fmt.Errorf("could not parse pid: %w", err)
		}
		return pid, err
	}
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

// CreateStandaloneAddressFile creates the standalone.txt file containing the address of the GRPC server
func CreateStandaloneAddressFile(info Args) error {
	return os.WriteFile(
		info.StandaloneAddressFilePath(),
		[]byte(info.Address),
		0600,
	)
}

// CreateStandalonePIDFile creates the pid.txt file containing the PID of the GRPC server process
func CreateStandalonePIDFile(info Args) error {
	return os.WriteFile(
		info.StandalonePIDFilePath(),
		[]byte(strconv.Itoa(os.Getpid())),
		0600,
	)
}

func cleanupStandaloneFile(fileName string) error {
	err := os.Remove(fileName)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// CleanupStandaloneAddressFile attempts to delete standalone.txt from the specified folder.
// If the file does not exist, the function returns nil.
func CleanupStandaloneAddressFile(info Args) error {
	return cleanupStandaloneFile(info.StandalonePIDFilePath())
}

// CleanupStandalonePIDFile attempts to delete pid.txt from the specified folder.
// If the file does not exist, the function returns nil.
func CleanupStandalonePIDFile(info Args) error {
	return cleanupStandaloneFile(info.StandalonePIDFilePath())
}

// FindAndKillCurrentPlugin kills the currently registered plugin, causing grafana to restart it
// this time pointing to our new host.
func FindAndKillCurrentPlugin(dir string) {
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

	out, err := exec.Command("pgrep", "-f", exeprefix).Output()
	if err != nil {
		fmt.Printf("error running pgrep: %s (%s)", err.Error(), exeprefix)
		return
	}
	currentPID := os.Getpid()
	for _, txt := range strings.Fields(string(out)) {
		pid, err := strconv.Atoi(txt)
		if err == nil {
			// Do not kill the plugin process
			if pid == currentPID {
				continue
			}
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

// CheckPIDIsRunning returns true if there's a process with the specified PID
func CheckPIDIsRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// FindProcess does not return an error if the process does not exist in UNIX.
	//
	// From man kill:
	//	 If  sig  is 0, then no signal is sent, but error checking is still per‐
	//   formed; this can be used to check for the existence of a process ID  or
	//   process group ID.
	//
	// So we send try to send a 0 signal to the process instead to test if it exists.
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false
	}
	return true
}
