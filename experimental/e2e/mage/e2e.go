package mage

import (
	"fmt"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e"
	ca "github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/certificate_authority"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/config"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/fixture"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/storage"
	"github.com/magefile/mage/mg"
)

// E2e is a namespace.
type E2e mg.Namespace

// Append starts the E2E proxy in append mode.
func (E2e) Append() error {
	return e2eProxy(e2e.ProxyModeAppend)
}

// Overwrite starts the E2E proxy in overwrite mode.
func (E2e) Overwrite() error {
	return e2eProxy(e2e.ProxyModeOverwrite)
}

// Replay starts the E2E proxy in replay mode.
func (E2e) Replay() error {
	return e2eProxy(e2e.ProxyModeReplay)
}

// Certificate prints the CA certificate to stdout.
func (E2e) Certificate() error {
	cfg, err := config.LoadConfig("proxy.json")
	if err != nil {
		return err
	}

	if cert, _, err := ca.LoadKeyPair(cfg.CAConfig.Cert, cfg.CAConfig.PrivateKey); err == nil {
		fmt.Print(string(cert))
		return nil
	}

	fmt.Print(string(ca.CACertificate))
	return nil
}

func e2eProxy(mode e2e.ProxyMode) error {
	cfg, err := config.LoadConfig("proxy.json")
	if err != nil {
		return err
	}
	fixtures := make([]*fixture.Fixture, 0)
	for _, s := range cfg.Storage {
		switch s.Type {
		case config.StorageTypeHAR:
			store := storage.NewHARStorage(s.Path)
			fixtures = append(fixtures, fixture.NewFixture(store))
		case config.StorageTypeOpenAPI:
			store := storage.NewOpenAPIStorage(s.Path)
			fixtures = append(fixtures, fixture.NewFixture(store))
		}
	}
	proxy := e2e.NewProxy(mode, fixtures, cfg)
	return proxy.Start()
}
