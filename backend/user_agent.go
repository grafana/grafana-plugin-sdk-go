package backend

type UserAgent struct {
	grafanaVersion string
	arch           string
	os             string
}

func NewUserAgent(grafanaVersion, arch, os string) *UserAgent {
	return &UserAgent{
		grafanaVersion: grafanaVersion,
		arch:           arch,
		os:             os,
	}
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

func FromString(s string) *UserAgent {
	return &UserAgent{
		grafanaVersion: s,
	}
}
