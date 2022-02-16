package e2e

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
	// ProxyModeOverwrite records new requests and responses.
	ProxyModeOverwrite
	// ProxyModeReplay replays existing responses if they match.
	ProxyModeReplay
)

// Proxy is a proxy server used for recording and replaying E2E test fixtures.
type Proxy struct {
	mode    ProxyMode
	Fixture *Fixture
	addr    string
	Server  *goproxy.ProxyHttpServer
}

// NewProxy creates a new Proxy.
func NewProxy(mode ProxyMode, fixture *Fixture, addr string) *Proxy {
	p := &Proxy{
		mode:    mode,
		Fixture: fixture,
		addr:    addr,
		Server:  goproxy.NewProxyHttpServer(),
	}
	p.Server.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	// Replay mode
	if p.mode == ProxyModeReplay {
		p.Server.OnRequest().DoFunc(p.replay)
		return p
	}

	// Append mode
	if p.mode == ProxyModeAppend {
		p.Server.OnRequest().DoFunc(p.replay)
		p.Server.OnResponse().DoFunc(p.append)
		return p
	}

	// Overwrite mode
	p.Server.OnRequest().DoFunc(p.request)
	p.Server.OnResponse().DoFunc(p.overwrite)
	return p
}

// Start starts the proxy server.
func (p *Proxy) Start() error {
	fmt.Println("Starting proxy", "mode", p.mode, "addr", p.addr)
	return http.ListenAndServe(p.addr, p.Server)
}

// request sends a request to the destination server.
func (p *Proxy) request(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	ctx.RoundTripper = goproxy.RoundTripperFunc(roundTripper)
	return req, nil
}

// replay returns a saved response for any matching request, and falls back to sending a request to the destination server.
func (p *Proxy) replay(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	ctx.RoundTripper = goproxy.RoundTripperFunc(roundTripper)
	if _, res := p.Fixture.Match(req); res != nil {
		return req, res
	}
	return req, nil
}

// append appends a response to the fixture store if there currently is not a match for the request.
func (p *Proxy) append(res *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if _, cached := p.Fixture.Match(res.Request); cached != nil {
		fmt.Println("Match", "url:", res.Request.URL.String(), "status:", res.StatusCode)
		return res
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
		fmt.Println("Removed existing match", "url:", res.Request.URL.String(), "status:", res.StatusCode)
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
