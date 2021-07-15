package grpcapi

import "encoding/json"

type PublishRequest struct {
	Channel string
	Data    json.RawMessage
}

// PublishResponse returned from Live server. This is empty at the moment,
// but can be extended with fields later.
type PublishResponse struct{}

// GetOrgTokenResult contains result with org token.
type GetOrgTokenResponse struct {
	Token string
}

// GetOrgTokenResult contains result with org token.
type GetOrgTokenRequest struct {
	OrgID int64
}
