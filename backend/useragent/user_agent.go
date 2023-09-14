package useragent

import (
	"errors"
	"regexp"
)

var (
	userAgentRegex   = regexp.MustCompile(`^Grafana/([0-9]+\.[0-9]+\.[0-9]+(?:-[a-zA-Z0-9]+)?) \(([a-zA-Z0-9]+); ([a-zA-Z0-9]+)\)$`)
	errInvalidFormat = errors.New("invalid user agent format")
)

// UserAgent represents a Grafana user agent.
// Its format is "Grafana/<version> (<os>; <arch>)"
// Example: "Grafana/7.0.0-beta1 (darwin; amd64)"
type UserAgent struct {
	grafanaVersion string
	arch           string
	os             string
}

func New(grafanaVersion, os, arch string) (*UserAgent, error) {
	ua := &UserAgent{
		grafanaVersion: grafanaVersion,
		os:             os,
		arch:           arch,
	}

	return NewFromString(ua.String())
}

func NewFromString(s string) (*UserAgent, error) {
	matches := userAgentRegex.FindStringSubmatch(s)
	if len(matches) != 4 {
		return nil, errInvalidFormat
	}

	return &UserAgent{
		grafanaVersion: matches[1],
		os:             matches[2],
		arch:           matches[3],
	}, nil
}

func (ua *UserAgent) GrafanaVersion() string {
	return ua.grafanaVersion
}

func (ua *UserAgent) Arch() string {
	return ua.arch
}

func (ua *UserAgent) OS() string {
	return ua.os
}

func (ua *UserAgent) String() string {
	return "Grafana/" + ua.grafanaVersion + " (" + ua.os + "; " + ua.arch + ")"
}
