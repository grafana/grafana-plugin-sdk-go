package e2eproxy

import (
	"bytes"
	"crypto/tls"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/elazarl/goproxy"
	"github.com/elazarl/goproxy/transport"
	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

type ProxyMode int

const (
	ProxyModeAppend   ProxyMode = iota
	ProxyModePlayback ProxyMode = iota
	ProxyModeRecord   ProxyMode = iota
)

func StartProxy(mode ProxyMode, fixture *Fixture, addr string) error {
	proxy := goproxy.NewProxyHttpServer()
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	if mode == ProxyModePlayback || mode == ProxyModeAppend {
		// ignoring the G402 error here because this proxy is only used for testing
		// nolint:gosec
		tr := transport.Transport{Proxy: transport.ProxyFromEnvironment, TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			ctx.RoundTripper = goproxy.RoundTripperFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (resp *http.Response, err error) {
				reqBody, err := ioutil.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				req.Body = ioutil.NopCloser(bytes.NewBuffer(reqBody))
				_, resp, err = tr.DetailedRoundTrip(req)
				if resp != nil {
					resp.Request.Body = io.NopCloser(bytes.NewBuffer(reqBody))
				}
				return resp, err
			})
			if res := fixture.Match(req); res != nil {
				return req, res
			}
			return req, nil
		})
	}

	if mode == ProxyModeRecord || mode == ProxyModeAppend {
		proxy.OnResponse().DoFunc(
			func(res *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
				if cached := fixture.Match(res.Request); cached != nil {
					backend.Logger.Debug("Proxy: matched response", "url", res.Request.URL.String(), "status", res.StatusCode)
					return res
				}
				fixture.Add(res.Request, res)
				err := fixture.Save()
				if err != nil {
					panic(err)
				}
				backend.Logger.Debug("Proxy: recorded response", "url", res.Request.URL.String(), "status", res.StatusCode)
				return res
			})
	}
	backend.Logger.Info("Starting proxy", "mode", mode, "addr", addr)
	return http.ListenAndServe(addr, proxy)
}
