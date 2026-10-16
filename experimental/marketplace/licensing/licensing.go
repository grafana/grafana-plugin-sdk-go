package licensing

import (
	"context"
	"encoding/json"

	"google.golang.org/grpc/metadata"
)

const upstreamLicenseHeader = "license"

// LicenseInfo is the license information supplied by Grafana on each request.
// Its metadata format matches the Grafana Enterprise SDK.
type LicenseInfo struct {
	LicenseToken   string `json:"licenseToken"`
	AppURL         string `json:"appURL"`
	ValidationKeys string `json:"validationKeys"`
}

// LicenseFromIncomingGRPCContext returns license info from incoming gRPC metadata.
func LicenseFromIncomingGRPCContext(ctx context.Context) (LicenseInfo, bool) {
	md, exists := metadata.FromIncomingContext(ctx)
	if !exists {
		return LicenseInfo{}, false
	}

	values := md.Get(upstreamLicenseHeader)
	if len(values) == 0 || values[0] == "" {
		return LicenseInfo{}, false
	}

	var info LicenseInfo
	if err := json.Unmarshal([]byte(values[0]), &info); err != nil {
		return LicenseInfo{}, false
	}
	return info, true
}
