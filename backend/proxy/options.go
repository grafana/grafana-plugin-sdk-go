package proxy

import "time"

// Options defines options for creating the proxy dialer.
type Options struct {
	Timeout   time.Duration
	KeepAlive time.Duration
}

// DefaultOptions default timeout/connection options for the proxy.
var DefaultOptions = Options{
	Timeout:   180 * time.Second,
	KeepAlive: 30 * time.Second,
}

func createOptions(providedOpts *Options) Options {
	if providedOpts == nil {
		return DefaultOptions
	}

	return *providedOpts
}
