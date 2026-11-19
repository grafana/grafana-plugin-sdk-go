package datasource

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"maps"
	"math/big"
	"runtime"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/grafana/grafana-plugin-sdk-go/backend/instancemgmt"
	"github.com/grafana/grafana-plugin-sdk-go/backend/log"
	"github.com/grafana/grafana-plugin-sdk-go/backend/proxy"
	"github.com/grafana/grafana-plugin-sdk-go/config"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/featuretoggles"
	"github.com/grafana/grafana-plugin-sdk-go/genproto/pluginv2"
)

type cacheShape struct {
	name    string
	toggles int
	pdc     bool
}

var cacheShapes = []cacheShape{
	{name: "toggles=5,pdc", toggles: 5, pdc: true},
	{name: "toggles=300,pdc", toggles: 300, pdc: true},
	{name: "toggles=5,no-pdc", toggles: 5},
}

func benchPEM(tb testing.TB, blockType string, der []byte) string {
	tb.Helper()
	return string(pem.EncodeToMemory(&pem.Block{Type: blockType, Bytes: der}))
}

func benchCertificate(tb testing.TB, name string) (cert, key string) {
	tb.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		tb.Fatal(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: name},
		DNSNames:     []string{name + ".example.com"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(90 * 24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &k.PublicKey, k)
	if err != nil {
		tb.Fatal(err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(k)
	if err != nil {
		tb.Fatal(err)
	}
	return benchPEM(tb, "CERTIFICATE", der), benchPEM(tb, "PRIVATE KEY", keyDER)
}

// benchStackConfigs returns one GrafanaConfig per stack. The toggle list and the PDC root CA are the same for every stack.
func benchStackConfigs(tb testing.TB, shape cacheShape, stacks int) []map[string]string {
	tb.Helper()
	toggles := []string{featuretoggles.TTLInstanceManager}
	for i := 1; i < shape.toggles; i++ {
		toggles = append(toggles, fmt.Sprintf("featureToggleName%03d", i))
	}
	rootCA, _ := benchCertificate(tb, "root-ca")
	cfgs := make([]map[string]string, stacks)
	for s := range cfgs {
		slug := fmt.Sprintf("stack%06d", s)
		cfg := map[string]string{
			config.AppURL:                  "https://" + slug + ".example.com/",
			config.ConcurrentQueryCount:    "10",
			featuretoggles.EnabledFeatures: strings.Join(toggles, ","),
			config.OpenFeatureProviderURL:  "http://feature-flags.example.svc:8080/",
			config.OpenFeatureProviderType: "ofrep",
			config.OpenFeatureCacheTTL:     "60",
			config.OpenFeatureContext:      `{"namespace":"stacks-` + slug + `","slug":"` + slug + `","stackId":"` + slug + `"}`,
			config.SQLRowLimit:             "1000000",
			config.SQLMaxOpenConnsDefault:  "100",
			config.SQLMaxIdleConnsDefault:  "100",
			config.ResponseLimit:           "0",
			config.UserFacingDefaultError:  "Please inspect the Grafana server log for details.",
		}
		if shape.pdc {
			cert, key := benchCertificate(tb, slug)
			cfg[proxy.PluginSecureSocksProxyEnabled] = "true"
			cfg[proxy.PluginSecureSocksProxyClientCert] = "/etc/pdc/client.crt"
			cfg[proxy.PluginSecureSocksProxyClientCertContents] = cert
			cfg[proxy.PluginSecureSocksProxyClientKey] = "/etc/pdc/client.key"
			cfg[proxy.PluginSecureSocksProxyClientKeyContents] = key
			cfg[proxy.PluginSecureSocksProxyRootCAs] = "/etc/pdc/ca.crt"
			cfg[proxy.PluginSecureSocksProxyRootCAsContents] = rootCA
			cfg[proxy.PluginSecureSocksProxyProxyAddress] = "pdc.example.com:443"
			cfg[proxy.PluginSecureSocksProxyServerName] = "pdc.example.com"
			cfg[proxy.PluginSecureSocksProxyAllowInsecure] = "false"
		}
		cfgs[s] = cfg
	}
	return cfgs
}

func benchWire(tb testing.TB, cfg map[string]string, uid string) []byte {
	tb.Helper()
	b, err := proto.Marshal(&pluginv2.PluginContext{
		OrgId:         1,
		PluginId:      "prometheus",
		GrafanaConfig: cfg,
		User:          &pluginv2.User{Login: "admin", Name: "Admin", Role: "Admin"},
		DataSourceInstanceSettings: &pluginv2.DataSourceInstanceSettings{
			Uid:                     uid,
			Name:                    "datasource-" + uid,
			Url:                     "https://metrics.example.com/api/prom",
			BasicAuthEnabled:        true,
			BasicAuthUser:           "123456",
			JsonData:                []byte(`{"httpMethod":"POST","timeInterval":"60s","timeout":60}`),
			DecryptedSecureJsonData: map[string]string{"basicAuthPassword": strings.Repeat("x", 120)},
			LastUpdatedMS:           1700000000000,
		},
	})
	if err != nil {
		tb.Fatal(err)
	}
	return b
}

func benchDecode(tb testing.TB, wire []byte) backend.PluginContext {
	tb.Helper()
	var p pluginv2.PluginContext
	if err := proto.Unmarshal(wire, &p); err != nil {
		tb.Fatal(err)
	}
	return backend.ConvertFromProtobuf{}.PluginContext(&p)
}

func benchGet(tb testing.TB, im instancemgmt.InstanceManager, pCtx backend.PluginContext) {
	tb.Helper()
	ctx := config.WithGrafanaConfig(context.Background(), pCtx.GrafanaConfig)
	if _, err := im.Get(ctx, pCtx); err != nil {
		tb.Fatal(err)
	}
}

func liveHeap() uint64 {
	runtime.GC()
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ms.HeapAlloc
}

func quietLogger(b *testing.B) {
	b.Helper()
	previous := backend.Logger
	backend.Logger = log.NewWithLevel(log.Info)
	b.Cleanup(func() { backend.Logger = previous })
}

func benchFactory(context.Context, backend.DataSourceInstanceSettings) (instancemgmt.Instance, error) {
	return struct{}{}, nil
}

var managerSink instancemgmt.InstanceManager

func BenchmarkInstanceCacheRetained(b *testing.B) {
	quietLogger(b)
	const instances = 1000
	for _, shape := range cacheShapes {
		for _, perStack := range []int{1, 4} {
			cfgs := benchStackConfigs(b, shape, instances/perStack)
			wires := make([][]byte, instances)
			for i := range wires {
				wires[i] = benchWire(b, cfgs[i/perStack], fmt.Sprintf("uid-%06d", i))
			}
			b.Run(fmt.Sprintf("%s/per-stack=%d", shape.name, perStack), func(b *testing.B) {
				var total float64
				var runs int
				for b.Loop() {
					im := NewInstanceManager(benchFactory)
					before := liveHeap()
					for _, w := range wires {
						benchGet(b, im, benchDecode(b, w))
					}
					total += float64(liveHeap()-before) / instances
					runs++
					managerSink = im
				}
				b.ReportMetric(total/float64(runs), "B/instance")
			})
		}
	}
}

func BenchmarkInstanceCacheGet(b *testing.B) {
	quietLogger(b)
	for _, shape := range cacheShapes {
		cfg := benchStackConfigs(b, shape, 1)[0]
		wire := benchWire(b, cfg, "uid-000001")
		requests := make([]backend.PluginContext, 64)
		for i := range requests {
			requests[i] = benchDecode(b, wire)
		}
		b.Run(shape.name, func(b *testing.B) {
			im := NewInstanceManager(benchFactory)
			benchGet(b, im, requests[0])
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				benchGet(b, im, requests[i%len(requests)])
				i++
			}
		})
	}
}

func BenchmarkInstanceCacheReplace(b *testing.B) {
	quietLogger(b)
	for _, shape := range cacheShapes {
		cfg := benchStackConfigs(b, shape, 1)[0]
		changed := maps.Clone(cfg)
		changed[config.ConcurrentQueryCount] = "11"
		requests := []backend.PluginContext{
			benchDecode(b, benchWire(b, cfg, "uid-000001")),
			benchDecode(b, benchWire(b, changed, "uid-000001")),
		}
		b.Run(shape.name, func(b *testing.B) {
			im := NewInstanceManager(benchFactory)
			b.ReportAllocs()
			i := 0
			for b.Loop() {
				benchGet(b, im, requests[i%2])
				i++
			}
		})
	}
}
