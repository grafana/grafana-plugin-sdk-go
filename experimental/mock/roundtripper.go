package mock

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/storage"
)

const (
	responseStatusNotImplemented = "501 Not implemented"
)

type RoundTripper struct {
	// Response mock
	Body            string
	FileName        string // filename (relative path of where it is being called)
	HARFileName     string // filename (relative path of where it is being called)
	StatusCode      int
	Status          string
	ResponseHeaders map[string]string
	// Authentication
	BasicAuthEnabled  bool
	BasicAuthUser     string
	BasicAuthPassword string
}

// RoundTrip provides a http transport method for simulating http response
// If HARFileName present, it will take priority
// Else if FileName present, it will read the response from the filename
// Else if Body present, it will echo the body
// Else default response {} will be sent
func (rt *RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	res := &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Body:       io.NopCloser(bytes.NewBufferString("{}")),
	}
	if rt.BasicAuthEnabled && (req.URL.User.String() != fmt.Sprintf("%s:%s", rt.BasicAuthUser, rt.BasicAuthPassword)) {
		res.StatusCode = 401
		res.Status = "401 Unauthorized"
		return res, nil
	}
	if rt.HARFileName != "" {
		storage := storage.NewHARStorage(rt.HARFileName)
		err := storage.Load()
		if err != nil {
			res.StatusCode = http.StatusNotImplemented
			res.Status = responseStatusNotImplemented
			res.Body = io.NopCloser(bytes.NewBufferString("no matching HAR files found"))
			return res, errors.New("no matching HAR files found")
		}
		matchedRequest := storage.Match(req)
		if matchedRequest == nil {
			res.StatusCode = http.StatusNotImplemented
			res.Status = responseStatusNotImplemented
			res.Body = io.NopCloser(bytes.NewBufferString("no matched request found in HAR file"))
			return res, errors.New("no matched request found in HAR file")
		}
		return matchedRequest, nil
	}
	if rt.FileName != "" {
		b, err := os.ReadFile(rt.FileName)
		if err != nil {
			return res, fmt.Errorf("error reading mock response file %s", rt.FileName)
		}
		reader := io.NopCloser(bytes.NewReader(b))
		defer reader.Close()
		res.Body = reader
		return rt.wrap(res), nil
	}
	if rt.Body != "" {
		res.Body = io.NopCloser(bytes.NewBufferString(rt.Body))
		return rt.wrap(res), nil
	}
	return rt.wrap(res), nil
}

func (rt *RoundTripper) wrap(res *http.Response) *http.Response {
	if rt.StatusCode != 0 {
		res.StatusCode = rt.StatusCode
	}
	if rt.Status != "" {
		res.Status = rt.Status
	}
	for key, value := range rt.ResponseHeaders {
		res.Header.Add(key, value)
	}
	return res
}

func GetMockHTTPClient(rt RoundTripper) *http.Client {
	h, _ := httpclient.New()
	h.Transport = &rt
	return h
}
