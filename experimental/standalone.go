package experimental

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/datasource"
)

type standaloneArgs struct {
	address    string
	standalone bool
}

// DoGRPC looks at the environment properties and decides if this should run as a normal hashicorp plugin or
// as a standalone gRPC server
func DoGRPC(id string, opts datasource.ServeOpts) error {
	// Enable profiler
	backend.SetupPluginEnvironment(id)

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

func getStandaloneInfo(id string) (standaloneArgs, error) {
	info := standaloneArgs{}

	var standalone bool
	var address string
	flag.BoolVar(&standalone, "standalone", false, "should this run standalone")
	flag.StringVar(&address, "address", "", "when running standalone this is the address")
	flag.Parse()

	info.standalone = standalone

	// standalone path
	ex, err := os.Executable()
	if err != nil {
		return info, err
	}

	// When debugging in vscode, write the file in `dist`
	if strings.HasSuffix(ex, "/pkg/__debug_bin") {
		ex = filepath.Join(filepath.Dir(ex), "..", "dist", "exe")
	}
	filePath := filepath.Join(filepath.Dir(ex), "standalone.txt")

	// Address from environment variable
	if address == "" {
		envvar := "GF_PLUGIN_GRPC_ADDRESS_" + strings.ReplaceAll(strings.ToUpper(id), "-", "_")
		address = os.Getenv(envvar)
	}

	// Check the local file for address
	addrBytes, err := ioutil.ReadFile(filePath)
	if address == "" {
		if err != nil && len(addrBytes) > 0 {
			address = string(addrBytes)
		}
	}
	info.address = address

	// Write the address to the local file
	if standalone {
		if info.address == "" {
			return info, fmt.Errorf("standalone address must be specified")
		}
		err = ioutil.WriteFile(filePath, []byte(info.address), 0600)
	}
	return info, err
}

func runDummyPluginLocator(address string) {
	fmt.Printf("1|2|tcp|%s|grpc\n", address)
	t := time.NewTicker(time.Second * 10)
	count := 0

	for range t.C {
		fmt.Printf("[%d] using address: %s\n", count, address)
		count++
	}

	// The hashicorp format is:
	// // Output the address and service name to stdout so that the client can bring it up.
	// fmt.Printf("%d|%d|%s|%s|%s|%s\n",
	// 	CoreProtocolVersion,
	// 	protoVersion,
	// 	listener.Addr().Network(),
	// 	listener.Addr().String(),
	// 	protoType,
	// 	serverCert)
}
