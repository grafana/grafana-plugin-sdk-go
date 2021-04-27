package backend

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend/httpclient"
)

// HTTPSettings convenient struct for holding decoded HTTP settings from
// jsonData and secureJSONData.
type HTTPSettings struct {
	Access            string `json:"access"`
	URL               string `json:"url"`
	BasicAuthEnabled  bool   `json:"basicAuth"`
	BasicAuthUser     string `json:"basicAuthUser"`
	BasicAuthPassword string `json:"secure.basicAuthPassword"`
	Headers           map[string]string

	Timeout               time.Duration `json:"timeout"`
	KeepAlive             time.Duration `json:"httpKeepAlive"`
	TLSHandshakeTimeout   time.Duration `json:"httpTLSHandshakeTimeout"`
	ExpectContinueTimeout time.Duration `json:"httpExpectContinueTimeout"`
	MaxIdleConns          int           `json:"httpMaxIdleConns"`
	MaxIdleConnsPerHost   int           `json:"httpMaxIdleConnsPerHost"`
	IdleConnTimeout       time.Duration `json:"httpIdleConnTimeout"`

	TLSClientAuth     bool   `json:"tlsAuth"`
	TLSAuthWithCACert bool   `json:"tlsAuthWithCACert"`
	TLSSkipVerify     bool   `json:"tlsSkipVerify"`
	TLSServerName     string `json:"serverName"`
	TLSCACert         string `json:"secure.tlsCACert"`
	TLSClientCert     string `json:"secure.tlsClientCert"`
	TLSClientKey      string `json:"secure.tlsClientKey"`

	SigV4Auth          bool   `json:"sigV4Auth"`
	SigV4Region        string `json:"sigV4Region"`
	SigV4AssumeRoleARN string `json:"sigV4AssumeRoleArn"`
	SigV4AuthType      string `json:"sigV4AuthType"`
	SigV4ExternalID    string `json:"sigV4ExternalId"`
	SigV4Profile       string `json:"sigV4Profile"`
	SigV4AccessKey     string `json:"secure.sigV4AccessKey"`
	SigV4SecretKey     string `json:"secure.sigV4SecretKey"`
}

// HTTPClientOptions creates and returns httpclient.Options.
func (s *HTTPSettings) HTTPClientOptions() httpclient.Options {
	opts := httpclient.Options{
		Headers: s.Headers,
	}

	opts.Timeouts = &httpclient.TimeoutOptions{
		Timeout:               s.Timeout,
		KeepAlive:             s.KeepAlive,
		TLSHandshakeTimeout:   s.TLSHandshakeTimeout,
		ExpectContinueTimeout: s.ExpectContinueTimeout,
		MaxIdleConns:          s.MaxIdleConns,
		MaxIdleConnsPerHost:   s.MaxIdleConnsPerHost,
		IdleConnTimeout:       s.IdleConnTimeout,
	}

	if s.BasicAuthEnabled {
		opts.BasicAuth = &httpclient.BasicAuthOptions{
			User:     s.BasicAuthUser,
			Password: s.BasicAuthPassword,
		}
	}

	if s.TLSClientAuth || s.TLSAuthWithCACert {
		opts.TLS = &httpclient.TLSOptions{
			CACertificate:      s.TLSCACert,
			ClientCertificate:  s.TLSClientCert,
			ClientKey:          s.TLSClientKey,
			InsecureSkipVerify: s.TLSSkipVerify,
			ServerName:         s.TLSServerName,
		}
	}

	if s.SigV4Auth {
		opts.SigV4 = &httpclient.SigV4Config{
			AuthType:      s.SigV4AuthType,
			Profile:       s.SigV4Profile,
			AccessKey:     s.SigV4AccessKey,
			SecretKey:     s.SigV4SecretKey,
			AssumeRoleARN: s.SigV4AssumeRoleARN,
			ExternalID:    s.SigV4ExternalID,
			Region:        s.SigV4Region,
		}
	}

	return opts
}

//gocyclo:ignore
func parseHTTPSettings(jsonData json.RawMessage, secureJSONData map[string]string) (*HTTPSettings, error) {
	s := &HTTPSettings{
		Headers: map[string]string{},
	}

	var dat map[string]interface{}
	if err := json.Unmarshal(jsonData, &dat); err != nil {
		return nil, err
	}

	if v, exists := dat["access"]; exists {
		s.Access = v.(string)
	} else {
		s.Access = "proxy"
	}

	if v, exists := dat["url"]; exists {
		s.URL = v.(string)
	}

	// Basic auth
	if v, exists := dat["basicAuth"]; exists {
		s.BasicAuthEnabled = v.(bool)
	}
	if s.BasicAuthEnabled {
		if v, exists := dat["basicAuthUser"]; exists {
			s.BasicAuthUser = v.(string)
		}
		if v, exists := secureJSONData["basicAuthPassword"]; exists {
			s.BasicAuthPassword = v
		}
	}

	// Timeouts
	if v, exists := dat["timeout"]; exists {
		if iv, ok := v.(float64); ok {
			s.Timeout = time.Duration(iv) * time.Second
		}
	} else {
		s.Timeout = httpclient.DefaultTimeoutOptions.Timeout
	}

	if v, exists := dat["httpKeepAlive"]; exists {
		if iv, ok := v.(float64); ok {
			s.KeepAlive = time.Duration(iv) * time.Second
		}
	} else {
		s.KeepAlive = httpclient.DefaultTimeoutOptions.KeepAlive
	}

	if v, exists := dat["httpTLSHandshakeTimeout"]; exists {
		if iv, ok := v.(float64); ok {
			s.TLSHandshakeTimeout = time.Duration(iv) * time.Second
		}
	} else {
		s.TLSHandshakeTimeout = httpclient.DefaultTimeoutOptions.TLSHandshakeTimeout
	}

	if v, exists := dat["httpExpectContinueTimeout"]; exists {
		if iv, ok := v.(float64); ok {
			s.ExpectContinueTimeout = time.Duration(iv) * time.Second
		}
	} else {
		s.ExpectContinueTimeout = httpclient.DefaultTimeoutOptions.ExpectContinueTimeout
	}

	if v, exists := dat["httpMaxIdleConns"]; exists {
		if iv, ok := v.(float64); ok {
			s.MaxIdleConns = int(iv)
		}
	} else {
		s.MaxIdleConns = httpclient.DefaultTimeoutOptions.MaxIdleConns
	}

	if v, exists := dat["httpMaxIdleConnsPerHost"]; exists {
		if iv, ok := v.(float64); ok {
			s.MaxIdleConnsPerHost = int(iv)
		}
	} else {
		s.MaxIdleConnsPerHost = httpclient.DefaultTimeoutOptions.MaxIdleConnsPerHost
	}

	if v, exists := dat["httpIdleConnTimeout"]; exists {
		if iv, ok := v.(float64); ok {
			s.IdleConnTimeout = time.Duration(iv) * time.Second
		}
	} else {
		s.IdleConnTimeout = httpclient.DefaultTimeoutOptions.IdleConnTimeout
	}

	// TLS
	if v, exists := dat["tlsAuth"]; exists {
		s.TLSClientAuth = v.(bool)
	}
	if v, exists := dat["tlsAuthWithCACert"]; exists {
		s.TLSAuthWithCACert = v.(bool)
	}

	if s.TLSClientAuth || s.TLSAuthWithCACert {
		if v, exists := dat["tlsSkipVerify"]; exists {
			s.TLSSkipVerify = v.(bool)
		}
		if v, exists := dat["serverName"]; exists {
			s.TLSServerName = v.(string)
		}
		if v, exists := secureJSONData["tlsCACert"]; exists {
			s.TLSCACert = v
		}
		if v, exists := secureJSONData["tlsClientCert"]; exists {
			s.TLSClientCert = v
		}
		if v, exists := secureJSONData["tlsClientKey"]; exists {
			s.TLSClientKey = v
		}
	}

	// SigV4
	if v, exists := dat["sigV4Auth"]; exists {
		s.SigV4Auth = v.(bool)
	}

	if s.SigV4Auth {
		if v, exists := dat["sigV4Region"]; exists {
			s.SigV4Region = v.(string)
		}
		if v, exists := dat["sigV4AssumeRoleArn"]; exists {
			s.SigV4AssumeRoleARN = v.(string)
		}
		if v, exists := dat["sigV4AuthType"]; exists {
			s.SigV4AuthType = v.(string)
		}
		if v, exists := dat["sigV4ExternalId"]; exists {
			s.SigV4ExternalID = v.(string)
		}
		if v, exists := dat["sigV4Profile"]; exists {
			s.SigV4Profile = v.(string)
		}
		if v, exists := secureJSONData["sigV4AccessKey"]; exists {
			s.SigV4AccessKey = v
		}
		if v, exists := secureJSONData["sigV4SecretKey"]; exists {
			s.SigV4SecretKey = v
		}
	}

	// headers
	index := 1
	for {
		headerNameSuffix := fmt.Sprintf("httpHeaderName%d", index)
		headerValueSuffix := fmt.Sprintf("httpHeaderValue%d", index)

		if key, exists := dat[headerNameSuffix]; exists {
			if value, exists := secureJSONData[headerValueSuffix]; exists {
				s.Headers[key.(string)] = value
			}
		} else {
			// No (more) header values are available
			break
		}
		index++
	}

	return s, nil
}
