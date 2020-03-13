package httpadapter

import (
	"net/http"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func Example() {
	handler := New(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("Hello world!"))
		rw.WriteHeader(http.StatusOK)
	}))
	_ = backend.ServeOpts{
		CallResourceHandler: handler,
	}
}

func Example_serve_mux() {
	mux := http.NewServeMux()
	mux.HandleFunc("/hello", func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("Hello world!"))
		rw.WriteHeader(http.StatusOK)
	})
	handler := New(mux)
	_ = backend.ServeOpts{
		CallResourceHandler: handler,
	}
}
