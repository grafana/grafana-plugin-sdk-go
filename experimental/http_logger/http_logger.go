package http_logger

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
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

type HttpLogger struct {
	enabled bool
	proxied http.RoundTripper
	fixture *fixture.Fixture
}

func NewHTTPLogger(pluginID string, proxied http.RoundTripper) *HttpLogger {
	if pluginID == "" {
		panic("pluginID cannot be empty")
	}

	path, ok := os.LookupEnv(PluginHARLogPathEnv)
	if !ok {
		path = createTemp(pluginID)
	}

	enabled := false
	if v, ok := os.LookupEnv(PluginHARLogEnabledEnv); ok && v == "true" {
		backend.Logger.Info("HTTP HAR Logging enabled", "pluginID", pluginID, "path", path)
		enabled = true
	}

	s := storage.NewHARStorage(path)
	if err := s.Load(); err != nil {
		s.Init()
	}

	f := fixture.NewFixture(s)

	return &HttpLogger{
		proxied: proxied,
		fixture: f,
		enabled: enabled,
	}
}

func (hl *HttpLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	if !hl.enabled {
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
	if _, exists := hl.fixture.Match(req); exists != nil {
		return res, err
	}

	hl.fixture.Add(req, res)
	err = hl.fixture.Save()

	return res, err
}

func createTemp(pluginID string) string {
	f, err := os.CreateTemp("", fmt.Sprintf("%s_%d_*.har", pluginID, time.Now().Unix()))
	if err != nil {
		panic("failed to create temporary file for HAR logging")
	}
	return f.Name()
}
