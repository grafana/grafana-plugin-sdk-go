package experimental

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/grpcplugin"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
	plugin "github.com/hashicorp/go-plugin"
)

type standaloneArgs struct {
	address    string
	standalone bool
}

func RunGRPC(id string, opts grpcplugin.ServeOpts) error {
	info, err := getStandaloneInfo(id)
	if err != nil {
		return err
	}
	if info.standalone {
		return runStandaloneServer(opts, info.address)
	} else if info.address != "" {
		runDummyPluginLocator(info.address)
		return nil
	}

	// The default/normal hashicorp path
	return grpcplugin.Serve(opts)
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
			info.address = string(addrBytes)
		}
	}

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

// runStandaloneServer starts a gRPC server that is not managed by hashicorp
func runStandaloneServer(opts grpcplugin.ServeOpts, address string) error {
	if opts.GRPCServer == nil {
		opts.GRPCServer = plugin.DefaultGRPCServer
	}

	server := opts.GRPCServer(nil)

	plugKeys := []string{}
	if opts.DiagnosticsServer != nil {
		pluginv2.RegisterDiagnosticsServer(server, opts.DiagnosticsServer)
		plugKeys = append(plugKeys, "diagnostics")
	}

	if opts.ResourceServer != nil {
		pluginv2.RegisterResourceServer(server, opts.ResourceServer)
		plugKeys = append(plugKeys, "resources")
	}

	if opts.DataServer != nil {
		pluginv2.RegisterDataServer(server, opts.DataServer)
		plugKeys = append(plugKeys, "data")
	}

	if opts.StreamServer != nil {
		pluginv2.RegisterStreamServer(server, opts.StreamServer)
		plugKeys = append(plugKeys, "stream")
	}

	log.DefaultLogger.Debug("Standalone plugin server", "capabilities", plugKeys)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	err = server.Serve(listener)
	if err != nil {
		return err
	}
	log.DefaultLogger.Debug("Plugin server exited")

	return nil
}
