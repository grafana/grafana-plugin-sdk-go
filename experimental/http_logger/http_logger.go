package httplogger

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/fixture"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/storage"
	"github.com/grafana/grafana-plugin-sdk-go/experimental/e2e/utils"
)

const (
	// PluginHARLogEnabledEnv is a constant for the GF_PLUGIN_HAR_LOG_ENABLED environment variable used to enable HTTP request and responses in HAR format for debugging purposes.
	PluginHARLogEnabledEnv = "GF_PLUGIN_HAR_LOG_ENABLED"
	// PluginHARLogPathEnv is a constant for the GF_PLUGIN_HAR_LOG_PATH environment variable used to specify a path to store HTTP request and responses in HAR format for debugging purposes.
	PluginHARLogPathEnv = "GF_PLUGIN_HAR_LOG_PATH"
)

// HTTPLogger is a http.RoundTripper that logs requests and responses in HAR format.
type HTTPLogger struct {
	pluginID string
	enabled  func() bool
	proxied  http.RoundTripper
	fixture  *fixture.Fixture
}

// NewHTTPLogger creates a new HTTPLogger.
func NewHTTPLogger(pluginID string, proxied http.RoundTripper) *HTTPLogger {
	path := defaultPath(pluginID)
	s := storage.NewHARStorage(path)
	f := fixture.NewFixture(s)

	return &HTTPLogger{
		pluginID: pluginID,
		proxied:  proxied,
		fixture:  f,
		enabled:  defaultEnabledCheck,
	}
}

// WithPath sets the path to store HAR file.
func (hl *HTTPLogger) WithPath(path string) *HTTPLogger {
	s := storage.NewHARStorage(path)
	hl.fixture = fixture.NewFixture(s)
	return hl
}

// WithEnabledCheck sets the function used to check if HTTP logging is enabled.
func (hl *HTTPLogger) WithEnabledCheck(fn func() bool) *HTTPLogger {
	hl.enabled = fn
	return hl
}

// RoundTrip implements the http.RoundTripper interface.
func (hl *HTTPLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	if !hl.enabled() {
		return hl.proxied.RoundTrip(req)
	}

	buf := []byte{}
	if req.Body != nil {
		if b, err := utils.ReadRequestBody(req); err == nil {
			req.Body = ioutil.NopCloser(bytes.NewReader(b))
			buf = b
		}
	}

	res, err := hl.proxied.RoundTrip(req)
	if err != nil {
		return res, err
	}

	// reset the request body before saving
	if req.Body != nil {
		req.Body = ioutil.NopCloser(bytes.NewBuffer(buf))
	}

	// skip saving if there's an existing entry for this request
	if exists := hl.fixture.Match(req); exists != nil {
		return res, err
	}

	hl.fixture.Add(req, res)
	err = hl.fixture.Save()

	return res, err
}

func defaultPath(pluginID string) string {
	if path, ok := os.LookupEnv(PluginHARLogPathEnv); ok {
		return path
	}
	return getTempFilePath(pluginID)
}

func defaultEnabledCheck() bool {
	if v, ok := os.LookupEnv(PluginHARLogEnabledEnv); ok && v == "true" {
		return true
	}
	return false
}

func getTempFilePath(pluginID string) string {
	filename := fmt.Sprintf("%s_%d.har", pluginID, time.Now().UnixMilli())
	return path.Join(os.TempDir(), filename)
}
