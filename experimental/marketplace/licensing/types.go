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

	Id                     string   `json:"jti"`
	Issuer                 string   `json:"iss"`
	Subject                string   `json:"sub"`
	Issued                 int64    `json:"iat"`
	Expires                int64    `json:"exp"`
	LicenseIssued          int64    `json:"nbf"`
	LicenseExpires         int64    `json:"lexp"`
	LicenseId              string   `json:"lid"`
	IncludedAdmins         int64    `json:"included_admins"`
	IncludedViewers        int64    `json:"included_viewers"`
	IncludedUsers          int64    `json:"included_users"`
	LicenseExpiresWarnDays int64    `json:"lic_exp_warn_days"`
	TokenExpiresWarnDays   int64    `json:"tok_exp_warn_days"`
	UpdateDays             int64    `json:"update_days"`
	Products               []string `json:"prod"`
	Company                string   `json:"company"`
	Slug                   string   `json:"slug"`
}

var (
	errInvalidAppURL = func(subject string) error {
		return fmt.Errorf("licensed URL '%s' is invalid, please contact support", subject)
	}
	errNoMatchAppURL = func(appURL, subject string) error {
		return fmt.Errorf("instance URL '%s' does not match licensed URL '%s'", appURL, subject)
	}
	errLicenseNotActiveYet = func(issuedAt time.Time) error {
		return fmt.Errorf("license issue date is %v", issuedAt.UTC())
	}
	errLicenseExpired = func(expiredAt time.Time) error {
		return fmt.Errorf("license expired at %v", expiredAt.UTC())
	}
	errTokenExpired = func(expiredAt time.Time) error {
		return fmt.Errorf("license token expired at %v", expiredAt.UTC())
	}

	errMarketplacePluginNotIncluded = errors.New("license does not include the plugin id as a product")
	ErrTokenNotFound                = errors.New("license token not found")
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
	if token.UpdateDays < 1 {
		token.UpdateDays = 1
	}

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
		token.Error = errLicenseNotActiveYet(time.Unix(token.LicenseIssued, 0))
		return false
	}

	if time.Unix(token.LicenseExpires, 0).Before(timeNow()) {
		token.Status = Expired
		token.Error = errLicenseExpired(time.Unix(token.LicenseExpires, 0))
		return false
	}

	if time.Unix(token.Expires, 0).Before(timeNow()) {
		token.Status = Expired
		token.Error = errTokenExpired(time.Unix(token.Expires, 0))
		return false
	}

	if _, ok := bLIDs[token.LicenseId]; ok {
		// for security purposes, we avoid returning that the licenses is blocked
		// we will return that a generic invalid license error
		token.Status = Invalid
		token.Error = errLicenseInvalid
		return false
	}

	found := false
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
	// older versions of Grafana don't provide appURL to plugins
	// so we have to skip the validation in that case
	// TODO: this should not be relevant for marketplace? They require latest Grafana.
	if appURL == "" {
		return nil
	}

	// if token subject is an hmac hash then appURL should also be a hash and we can just compare them directly
	if strings.HasPrefix(token.Subject, "hmac:") {
		if subtle.ConstantTimeCompare([]byte(token.Subject), []byte(appURL)) != 1 {
			return errNoMatchAppURL(appURL, token.Subject)
		}

		return nil
	}

	g, err := glob.Compile(token.Subject, '.', '/', ':')
	if err != nil {
		return errInvalidAppURL(token.Subject)
	}

	if !g.Match(appURL) {
		return errNoMatchAppURL(appURL, token.Subject)
	}

	return nil
}
