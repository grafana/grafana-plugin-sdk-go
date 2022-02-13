package e2eproxy

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/transport"
)

// ProxyMode is the record or playback mode of the Proxy.
type ProxyMode int

const (
	// ProxyModeAppend records new requests and responses, and replays existing responses if they match.
	ProxyModeAppend ProxyMode = iota
	// ProxyModeRecord records new requests and responses.
	ProxyModeRecord
	// ProxyModePlayback replays existing responses if they match.
	ProxyModePlayback
)

// Proxy is a proxy server used for recording and replaying E2E test fixtures.
type Proxy struct {
	mode    ProxyMode
	fixture *Fixture
	addr    string
	proxy   *goproxy.ProxyHttpServer
}

// NewProxy creates a new Proxy.
func NewProxy(mode ProxyMode, fixture *Fixture, addr string) *Proxy {
	return &Proxy{
		mode:    mode,
		fixture: fixture,
		addr:    addr,
		proxy:   goproxy.NewProxyHttpServer(),
	}
}

// Start starts the proxy server.
func (p *Proxy) Start() error {
	p.proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	if p.mode == ProxyModePlayback || p.mode == ProxyModeAppend {
		p.proxy.OnRequest().DoFunc(p.onRequest)
	}
	if p.mode == ProxyModeRecord || p.mode == ProxyModeAppend {
		p.proxy.OnResponse().DoFunc(p.onResponse)
	}
	fmt.Println("Starting proxy", "mode", p.mode, "addr", p.addr)
	return http.ListenAndServe(p.addr, p.proxy)
}

func (p *Proxy) onRequest(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	ctx.RoundTripper = goproxy.RoundTripperFunc(roundTripper)
	if res := p.fixture.Match(req); res != nil {
		return req, res
	}
	return req, nil
}

func (p *Proxy) onResponse(res *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if cached := p.fixture.Match(res.Request); cached != nil {
		fmt.Println("Match", "url:", res.Request.URL.String(), "status:", res.StatusCode)
		return res
	}
	p.fixture.Add(res.Request, res)
	err := p.fixture.Save()
	if err != nil {
		panic(err)
	}
	fmt.Println("Record", "url:", res.Request.URL.String(), "status:", res.StatusCode)
	return res
}

func roundTripper(req *http.Request, ctx *goproxy.ProxyCtx) (resp *http.Response, err error) {
	// ignoring the G402 error here because this proxy is only used for testing
	// nolint:gosec
	tr := transport.Transport{
		Proxy: transport.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	buf := &bytes.Buffer{}
	tee := io.TeeReader(req.Body, buf)
	req.Body = ioutil.NopCloser(tee)
	_, resp, err = tr.DetailedRoundTrip(req)
	if resp != nil {
		resp.Request.Body = io.NopCloser(buf)
	}
	return resp, err
}
