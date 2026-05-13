package licensing

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gobwas/glob"
	jose "gopkg.in/go-jose/go-jose.v2"
)

type TokenStatus int

var (
	timeNow = time.Now
	// map for blocked licenses ids. We use this as indexed array, the value of each key is ignored
	// this should be in sync with grafana-enterprise
	bLIDs = map[string]struct{}{}
)

const (
	NotLoaded TokenStatus = iota
	Valid
	Loaded
	Invalid
	NotFound
	Expired
	InvalidSubject
)

type LicenseToken struct {
	Raw    string      `json:"-"`
	Status TokenStatus `json:"status"`
	Error  error       `json:"-"`

	Id             string `json:"jti"`
	Issuer         string `json:"iss"`
	Subject        string `json:"sub"`
	Issued         int64  `json:"iat"`
	Expires        int64  `json:"exp"`
	LicenseIssued  int64  `json:"nbf"`
	LicenseExpires int64  `json:"lexp"`
	LicenseId      string `json:"lid"`

	// TODO: usage tracking?
	IncludedAdmins  int64 `json:"included_admins"`
	IncludedViewers int64 `json:"included_viewers"`
	IncludedUsers   int64 `json:"included_users"`

	// TODO: not in design doc?
	// LicenseExpiresWarnDays int64  `json:"lic_exp_warn_days"`
	// TokenExpiresWarnDays   int64  `json:"tok_exp_warn_days"`
	// UpdateDays             int64    `json:"update_days"`

	Products []string `json:"prod"`
	Company  string   `json:"company"`
	// TODO: not in design doc?
	// Account string `json:"account,omitempty"`
	Slug string `json:"slug"`
}

var (
	errInvalidAppURL                = errors.New("invalid licensed URL, please contact support")
	errNoMatchAppURL                = errors.New("instance URL does not match licensed URL")
	errLicenseNotActiveYet          = errors.New("license is not active yet")
	errLicenseExpired               = errors.New("license expired")
	errTokenExpired                 = errors.New("license token expired")
	errMarketplacePluginNotIncluded = errors.New("license does not include the marketplace plugin id as a product")
	// generic error
	errLicenseInvalid = errors.New("invalid license")
)

func (token *LicenseToken) Parse(tokenStr, appUrl, validationKeys, pluginId string) bool {
	token.Raw = tokenStr

	parsed, err := jose.ParseSigned(token.Raw)
	if err != nil {
		token.Status = Invalid
		token.Error = fmt.Errorf("%w: %w", errParsing, err)
		return false
	}

	keys, err := keySet(validationKeys)
	if err != nil {
		token.Status = Invalid
		token.Error = err
		return false
	}

	payload, err := unwrapSignedJWT(keys, parsed)
	if err != nil {
		token.Status = Invalid
		token.Error = err
		return false
	}

	err = json.Unmarshal(payload, &token)
	if err != nil {
		token.Status = Invalid
		token.Error = fmt.Errorf("%w: %w", errParsing, err)
		return false
	}

	// Handle tokens with missing or invalid "update_days" field
	// TODO: is this needed for marketplace?
	/* if token.UpdateDays < 1 {
		token.UpdateDays = 1
	} */

	logger.Debug("license token parsed", "token", token)
	return token.validate(appUrl, pluginId)
}

// validate validates the license against our licensing rules.
func (token *LicenseToken) validate(appURL, pluginId string) bool {
	if err := token.validateSubject(appURL); err != nil {
		token.Status = InvalidSubject
		token.Error = err
		return false
	}

	if time.Unix(token.LicenseIssued, 0).After(timeNow()) {
		token.Status = Invalid
		token.Error = fmt.Errorf("%w: license not active until %v", errLicenseNotActiveYet, time.Unix(token.LicenseIssued, 0))
		return false
	}

	if time.Unix(token.LicenseExpires, 0).Before(timeNow()) {
		token.Status = Expired
		token.Error = fmt.Errorf("%w: license expired at %v", errLicenseExpired, time.Unix(token.LicenseExpires, 0))
		return false
	}

	if time.Unix(token.Expires, 0).Before(timeNow()) {
		token.Status = Expired
		token.Error = fmt.Errorf("%w: token expired at %v", errTokenExpired, time.Unix(token.Expires, 0))
		return false
	}

	if _, ok := bLIDs[token.LicenseId]; ok {
		// for security purposes, we avoid returning that the licenses is blocked
		// we will return that a generic invalid license error
		token.Status = Invalid
		token.Error = errLicenseInvalid
		return false
	}

	var found bool
	for _, product := range token.Products {
		if pluginId != "" && product == "marketplace-"+pluginId {
			found = true
			break
		}
	}
	if !found {
		token.Status = Invalid
		token.Error = errMarketplacePluginNotIncluded
		return false
	}

	token.Status = Valid
	token.Error = nil
	return true
}

// validateSubject validates the licensed url
func (token *LicenseToken) validateSubject(appURL string) error {
	if appURL == "" {
		return fmt.Errorf("%w: %q", errInvalidAppURL, token.Subject)
	}

	// if token subject is an hmac hash then appURL should also be a hash and we can just compare them directly
	if strings.HasPrefix(token.Subject, "hmac:") {
		if subtle.ConstantTimeCompare([]byte(token.Subject), []byte(appURL)) != 1 {
			return fmt.Errorf("%w: instance %q, license %q", errNoMatchAppURL, appURL, token.Subject)
		}

		return nil
	}

	g, err := glob.Compile(token.Subject, '.', '/', ':')
	if err != nil {
		return fmt.Errorf("%w: %q", errInvalidAppURL, token.Subject)
	}

	if !g.Match(appURL) {
		return fmt.Errorf("%w: instance %q, license %q", errNoMatchAppURL, appURL, token.Subject)
	}

	return nil
}
