package e2eproxy

import (
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"gopkg.in/elazarl/goproxy.v1"
)

type ProxyMode int

const (
	ProxyModeAppend   ProxyMode = iota
	ProxyModePlayback ProxyMode = iota
	ProxyModeRecord   ProxyMode = iota
)

func StartProxy(mode ProxyMode, fixture *Fixture, addr string) *goproxy.ProxyHttpServer {
	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true
	if err := http.ListenAndServe(addr, proxy); err != nil {
		panic(err)
	}
	backend.Logger.Debug("Proxy started", "mode", mode, "addr", addr)

	if mode == ProxyModePlayback || mode == ProxyModeAppend {
		proxy.OnRequest().DoFunc(
			func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
				if res := fixture.Match(req); res != nil {
					return req, res
				}
				return req, nil
			})
	}

	if mode == ProxyModeRecord || mode == ProxyModeAppend {
		proxy.OnResponse().DoFunc(
			func(res *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
				fixture.store.Add(ctx.Req, res)
				return res
			})
	}

	return proxy
}
