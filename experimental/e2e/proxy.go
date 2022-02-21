package e2e

import (
	"crypto/tls"
	"crypto/x509"
	_ "embed" // used for embedding the CA certificate and key
	"fmt"
	"net/http"

	"github.com/elazarl/goproxy"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/config"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/fixture"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/utils"
)

// ProxyMode is the record or playback mode of the Proxy.
type ProxyMode int

func (m ProxyMode) String() string {
	switch m {
	case ProxyModeReplay:
		return "replay"
	case ProxyModeAppend:
		return "append"
	case ProxyModeOverwrite:
		return "overwrite"
	default:
		return "unknown"
	}
}

const (
	// ProxyModeAppend records new requests and responses, and replays existing responses if they match.
	ProxyModeAppend ProxyMode = iota
	// ProxyModeOverwrite records new requests and responses.
	ProxyModeOverwrite
	// ProxyModeReplay replays existing responses if they match.
	ProxyModeReplay
)

// Proxy is a proxy server used for recording and replaying E2E test fixtures.
type Proxy struct {
	Mode    ProxyMode
	Fixture *fixture.Fixture
	Server  *goproxy.ProxyHttpServer
	Config  *config.Config
}

// NewProxy creates a new Proxy.
func NewProxy(mode ProxyMode, fixture *fixture.Fixture, config *config.Config) *Proxy {
	err := setupCA()
	if err != nil {
		panic(err)
	}

	p := &Proxy{
		Mode:    mode,
		Fixture: fixture,
		Server:  goproxy.NewProxyHttpServer(),
		Config:  config,
	}

	reqConditions := []goproxy.ReqCondition{}
	respConditions := []goproxy.RespCondition{}
	for _, h := range config.Hosts {
		reqConditions = append(reqConditions, goproxy.DstHostIs(h))
		respConditions = append(respConditions, goproxy.RespConditionFunc(func(resp *http.Response, ctx *goproxy.ProxyCtx) bool {
			return resp.Request.URL.Host == h
		}))
	}

	p.Server.OnRequest(reqConditions...).HandleConnect(goproxy.AlwaysMitm)

	// Replay mode
	if p.Mode == ProxyModeReplay {
		p.Server.OnRequest(reqConditions...).DoFunc(p.replay)
		return p
	}

	// Append mode
	if p.Mode == ProxyModeAppend {
		p.Server.OnRequest(reqConditions...).DoFunc(p.replay)
		p.Server.OnResponse(respConditions...).DoFunc(p.append)
		return p
	}

	// Overwrite mode
	p.Server.OnRequest(reqConditions...).DoFunc(p.request)
	p.Server.OnResponse(respConditions...).DoFunc(p.overwrite)
	return p
}

// Start starts the proxy server.
func (p *Proxy) Start() error {
	fmt.Println("Starting proxy", "mode", p.Mode.String(), "addr", p.Config.Address)
	return http.ListenAndServe(p.Config.Address, p.Server)
}

// request sends a request to the destination server.
func (p *Proxy) request(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	ctx.RoundTripper = goproxy.RoundTripperFunc(utils.RoundTripper)
	return req, nil
}

// replay returns a saved response for any matching request, and falls back to sending a request to the destination server.
func (p *Proxy) replay(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	ctx.RoundTripper = goproxy.RoundTripperFunc(utils.RoundTripper)
	if _, res := p.Fixture.Match(req); res != nil {
		return req, res
	}
	return req, nil
}

// append appends a response to the fixture store if there currently is not a match for the request.
func (p *Proxy) append(res *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if _, cached := p.Fixture.Match(res.Request); cached != nil {
		fmt.Println("Match", "url:", cached.Request.URL.String(), "status:", cached.StatusCode)
		return cached
	}
	p.Fixture.Add(res.Request, res)
	err := p.Fixture.Save()
	if err != nil {
		panic(err)
	}
	fmt.Println("Append", "url:", res.Request.URL.String(), "status:", res.StatusCode)
	return res
}

// overwrite replaces a response in the fixture store if there currently is a match for the request.
func (p *Proxy) overwrite(res *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if id, cached := p.Fixture.Match(res.Request); cached != nil {
		fmt.Println("Removed existing match", "url:", cached.Request.URL.String(), "status:", cached.StatusCode)
		p.Fixture.Delete(id)
	}
	p.Fixture.Add(res.Request, res)
	err := p.Fixture.Save()
	if err != nil {
		panic(err)
	}
	fmt.Println("Overwrite", "url:", res.Request.URL.String(), "status:", res.StatusCode)
	return res
}

//go:embed cert/grafana-e2e-ca.pem
// CACertificate Certificate Authority certificate used by the proxy.
var CACertificate []byte

//go:embed cert/grafana-e2e-ca.key.pem
// CAKey Certificate Authority private key used by the proxy.
var CAKey []byte

func setupCA() error {
	goproxyCa, err := tls.X509KeyPair(CACertificate, CAKey)
	if err != nil {
		return err
	}
	if goproxyCa.Leaf, err = x509.ParseCertificate(goproxyCa.Certificate[0]); err != nil {
		return err
	}
	goproxy.GoproxyCa = goproxyCa
	goproxy.OkConnect = &goproxy.ConnectAction{Action: goproxy.ConnectAccept, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.MitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.HTTPMitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectHTTPMitm, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	goproxy.RejectConnect = &goproxy.ConnectAction{Action: goproxy.ConnectReject, TLSConfig: goproxy.TLSConfigFromCA(&goproxyCa)}
	return nil
}
