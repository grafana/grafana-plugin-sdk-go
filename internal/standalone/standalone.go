package standalone

import (
	"errors"
	"flag"
	"fmt"
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
	PID        int
	Standalone bool
	Dir        string
	Debugger   bool
}

// StandaloneAddressFilePath returns the path to the standalone.txt file
func (a Args) StandaloneAddressFilePath() string {
	return filepath.Join(a.Dir, "standalone.txt")
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
		port, err := getFreePort()
		if err == nil {
			info.Address = fmt.Sprintf(":%d", port)
		}
		js, err := findPluginJSON(ex)
		if err != nil {
			return info, err
		}
		ex = js
	}
	info.Dir = filepath.Dir(ex)
	filePath := info.StandaloneAddressFilePath()

	// Address from environment variable
	if info.Address == "" {
		envvar := "GF_PLUGIN_GRPC_ADDRESS_" + strings.ReplaceAll(strings.ToUpper(id), "-", "_")
		info.Address = os.Getenv(envvar)
	}

	if info.Address != "" {
		return info, nil
	}

	// Check the local file for address
	// Format:
	//
	//	:XYZ
	//	PID
	//
	// (PID is optional)
	standaloneFileContent, err := os.ReadFile(filePath)
	switch {
	case err != nil && !os.IsNotExist(err):
		return info, fmt.Errorf("read standalone file: %w", err)
	case os.IsNotExist(err) || len(standaloneFileContent) == 0:
		// No standalone file, do not treat as standalone
		return info, nil
	}
	parts := strings.Split(string(standaloneFileContent), "\n")
	if len(parts) < 1 {
		return info, errors.New("invalid standalone file content")
	}
	info.Address = parts[0]
	if len(parts) >= 2 {
		// Read PID (optional)
		pid, err := strconv.Atoi(parts[1])
		if err != nil {
			return info, fmt.Errorf("could not parse pid: %w", err)
		}
		info.PID = pid
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

// CreateStandaloneAddressFile creates the standalone.txt file containing the address of the GRPC server
// and the PID of the process.
func CreateStandaloneAddressFile(info Args) error {
	return os.WriteFile(
		info.StandaloneAddressFilePath(),
		[]byte(info.Address+"\n"+strconv.Itoa(os.Getpid())),
		0600,
	)
}

// CleanupStandaloneAddressFile attempts to delete standalone.txt from the specified folder.
// If the file does not exist, the function returns nil.
func CleanupStandaloneAddressFile(info Args) error {
	err := os.Remove(info.StandaloneAddressFilePath())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
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
	if _, err := os.FindProcess(pid); err != nil {
		return false
	}
	return true
}
