package licensing

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLicenseTokenValidation(t *testing.T) {
	bLicenses := map[string]struct{}{}
	fixedTestTime(t, time.Date(2026, 06, 13, 0, 0, 0, 0, time.UTC))

	tests := []struct {
		name           string
		token          *LicenseToken
		appURL         string
		bLicenses      map[string]struct{}
		status         TokenStatus
		expErr         error
		expErrContains string
	}{
		{
			name: "When the license is not active yet",
			token: &LicenseToken{
				Subject:       appURL,
				LicenseIssued: timeNow().Unix() + 60,
			},
			appURL:         appURL,
			bLicenses:      bLicenses,
			status:         Invalid,
			expErr:         errLicenseNotActiveYet,
			expErrContains: time.Unix(timeNow().Add(1*time.Minute).Unix(), 0).String(),
		},
		{
			name: "When the license has expired",
			token: &LicenseToken{
				Subject:        appURL,
				LicenseIssued:  timeNow().Unix(),
				LicenseExpires: timeNow().Unix() - 60,
				Products:       []string{testProductID},
			},
			appURL:         appURL,
			bLicenses:      bLicenses,
			status:         ExpiredInGracePeriod,
			expErr:         errLicenseExpired,
			expErrContains: time.Unix(timeNow().Add(-1*time.Minute).Unix(), 0).String(),
		},
		{
			name: "When the token has expired",
			token: &LicenseToken{
				Subject:        appURL,
				LicenseIssued:  timeNow().Unix(),
				LicenseExpires: timeNow().Unix(),
				Expires:        timeNow().Unix() - 60,
				Products:       []string{testProductID},
			},
			appURL:         appURL,
			bLicenses:      bLicenses,
			status:         ExpiredInGracePeriod,
			expErr:         errTokenExpired,
			expErrContains: time.Unix(timeNow().Add(-1*time.Minute).Unix(), 0).String(),
		},
		{
			name: "With a non-matching product",
			token: &LicenseToken{
				Subject:        appURL,
				LicenseIssued:  timeNow().Unix(),
				LicenseExpires: timeNow().Unix(),
				Expires:        timeNow().Unix(),
			},
			appURL:    appURL,
			bLicenses: bLicenses,
			status:    Invalid,
			expErr:    errMarketplacePluginNotIncluded,
		},
		{
			name: "With a non-matching url",
			token: &LicenseToken{
				Subject:        "http://localhost:3000/",
				LicenseIssued:  timeNow().Unix(),
				LicenseExpires: timeNow().Unix(),
				Expires:        timeNow().Unix(),
				Products:       []string{testProductID},
			},
			appURL:         appURL,
			bLicenses:      bLicenses,
			status:         Invalid,
			expErr:         errNoMatchAppURL,
			expErrContains: fmt.Sprintf("instance %q, license %q", appURL, "http://localhost:3000/"),
		},
		{
			name: "With a valid token",
			token: &LicenseToken{
				Subject:        appURL,
				LicenseIssued:  timeNow().Unix(),
				LicenseExpires: timeNow().Unix(),
				Expires:        timeNow().Unix(),
				Products:       []string{testProductID},
			},
			appURL:    appURL,
			bLicenses: bLicenses,
			status:    Valid,
			expErr:    nil,
		},
		{
			name: "With a glob instance url",
			token: &LicenseToken{
				Subject:        "http*://grafana*.mycompany.com",
				LicenseIssued:  timeNow().Unix(),
				LicenseExpires: timeNow().Unix(),
				Expires:        timeNow().Unix(),
				Products:       []string{testProductID},
			},
			appURL:    appURL,
			bLicenses: bLicenses,
			status:    Valid,
			expErr:    nil,
		},
		{
			name: "With a hashed instance url",
			token: &LicenseToken{
				Subject:        "hmac:test",
				LicenseIssued:  timeNow().Unix(),
				LicenseExpires: timeNow().Unix(),
				Expires:        timeNow().Unix(),
				Products:       []string{testProductID},
			},
			appURL:    "hmac:test",
			bLicenses: bLicenses,
			status:    Valid,
			expErr:    nil,
		},
		{
			name: "Without an instance url",
			token: &LicenseToken{
				Subject:        appURL,
				LicenseIssued:  timeNow().Unix(),
				LicenseExpires: timeNow().Unix(),
				Expires:        timeNow().Unix(),
				Products:       []string{testProductID},
			},
			appURL:    "",
			bLicenses: bLicenses,
			status:    Invalid,
			expErr:    errInvalidAppURL,
		},
		{
			name: "With a blocked license ID",
			token: &LicenseToken{
				LicenseId:      "12345",
				Subject:        appURL,
				LicenseIssued:  timeNow().Unix(),
				LicenseExpires: timeNow().Unix(),
				Expires:        timeNow().Unix(),
				Products:       []string{testProductID},
			},
			appURL:    appURL,
			bLicenses: map[string]struct{}{"12345": {}},
			status:    Invalid,
			expErr:    errLicenseInvalid,
		},
		{
			name: "With a non-matching product and an expired license",
			token: &LicenseToken{
				Subject:        appURL,
				LicenseIssued:  timeNow().Unix(),
				LicenseExpires: timeNow().Unix() - 60,
				Expires:        timeNow().Unix(),
				Products:       []string{"marketplace-other-plugin"},
			},
			appURL:    appURL,
			bLicenses: bLicenses,
			status:    Invalid,
			expErr:    errMarketplacePluginNotIncluded,
		},
		{
			name: "With a non-matching product and an expired token",
			token: &LicenseToken{
				Subject:        appURL,
				LicenseIssued:  timeNow().Unix(),
				LicenseExpires: timeNow().Unix(),
				Expires:        timeNow().Unix() - 60,
				Products:       []string{"marketplace-other-plugin"},
			},
			appURL:    appURL,
			bLicenses: bLicenses,
			status:    Invalid,
			expErr:    errMarketplacePluginNotIncluded,
		},
		{
			name: "With a blocked license ID and an expired license",
			token: &LicenseToken{
				LicenseId:      "12345",
				Subject:        appURL,
				LicenseIssued:  timeNow().Unix(),
				LicenseExpires: timeNow().Unix() - 60,
				Expires:        timeNow().Unix(),
				Products:       []string{testProductID},
			},
			appURL:    appURL,
			bLicenses: map[string]struct{}{"12345": {}},
			status:    Invalid,
			expErr:    errLicenseInvalid,
		},
		{
			name: "With a blocked license ID and an expired token",
			token: &LicenseToken{
				LicenseId:      "12345",
				Subject:        appURL,
				LicenseIssued:  timeNow().Unix(),
				LicenseExpires: timeNow().Unix(),
				Expires:        timeNow().Unix() - 60,
				Products:       []string{testProductID},
			},
			appURL:    appURL,
			bLicenses: map[string]struct{}{"12345": {}},
			status:    Invalid,
			expErr:    errLicenseInvalid,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.bLicenses) != 0 {
				useBlockedLicenseIds(t, tc.bLicenses)
			}

			ok := tc.token.validate(tc.appURL, testPluginID)

			if tc.expErr != nil {
				require.Equal(t, false, ok)
				require.Equal(t, tc.status, tc.token.Status)
				require.Error(t, tc.token.Error)
				require.ErrorIs(t, tc.token.Error, tc.expErr)
				if errors.Is(tc.expErr, errLicenseInvalid) {
					require.Equal(t, errLicenseInvalid, tc.token.Error)
				}
				if tc.expErrContains != "" {
					require.Contains(t, tc.token.Error.Error(), tc.expErrContains)
				}
				return
			}

			assert.Equal(t, true, ok, "validate should return true")
			assert.Equal(t, Valid, tc.token.Status)
			assert.NoError(t, tc.token.Error)
		})
	}
}

func useBlockedLicenseIds(tb testing.TB, bl map[string]struct{}) {
	tb.Helper()
	origLicenses := bLIDs
	bLIDs = bl
	tb.Cleanup(func() {
		bLIDs = origLicenses
	})
}

func TestAccepted(t *testing.T) {
	for status, want := range map[TokenStatus]bool{
		NotLoaded:            false,
		Valid:                true,
		ExpiredInGracePeriod: true,
		Expired:              false,
		Invalid:              false,
		NotFound:             false,
	} {
		token := &LicenseToken{Status: status, LicenseExpires: time.Now().Unix()}
		require.Equalf(t, want, token.Accepted(), "status %d", status)
	}
}

func TestGraceDeadlineUsesLicenseExpiration(t *testing.T) {
	licenseExpires := time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
	token := &LicenseToken{Expires: licenseExpires.Add(-30 * 24 * time.Hour).Unix(), LicenseExpires: licenseExpires.Unix()}
	require.True(t, licenseExpires.Add(GracePeriod).Equal(token.GraceDeadline()))
}

func TestValidateExpiredTokenGracePeriod(t *testing.T) {
	now := time.Date(2025, time.January, 15, 12, 0, 0, 0, time.UTC)
	fixedTestTime(t, now)

	tests := []struct {
		name                    string
		expires, licenseExpires time.Time
		status                  TokenStatus
		err                     error
	}{
		{"license expired within grace period", now.Add(time.Hour), now.Add(-24 * time.Hour), ExpiredInGracePeriod, errLicenseExpired},
		{"license expired just before the grace deadline", now.Add(time.Hour), now.Add(-GracePeriod + time.Second), ExpiredInGracePeriod, errLicenseExpired},
		{"license expired at the grace deadline", now.Add(time.Hour), now.Add(-GracePeriod), Expired, errLicenseExpired},
		{"license expired after the grace period", now.Add(time.Hour), now.Add(-30 * 24 * time.Hour), Expired, errLicenseExpired},
		{"token expired with an active license", now.Add(-time.Hour), now.Add(24 * time.Hour), ExpiredInGracePeriod, errTokenExpired},
		{
			// On-prem instances that lose internet connection cannot renew their short-lived token,
			// so the grace period follows the license expiration, not the token expiration.
			name:    "old token expiration within the license grace period",
			expires: now.Add(-30 * 24 * time.Hour), licenseExpires: now.Add(-24 * time.Hour),
			status: ExpiredInGracePeriod, err: errLicenseExpired,
		},
		{
			name:    "recent token expiration after the license grace period",
			expires: now.Add(-24 * time.Hour), licenseExpires: now.Add(-30 * 24 * time.Hour),
			status: Expired, err: errLicenseExpired,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token := &LicenseToken{
				Subject:        appURL,
				Products:       []string{testProductID},
				LicenseIssued:  now.Add(-60 * 24 * time.Hour).Unix(),
				Expires:        tc.expires.Unix(),
				LicenseExpires: tc.licenseExpires.Unix(),
			}
			require.False(t, token.validate(appURL, testPluginID))
			require.Equal(t, tc.status, token.Status)
			require.True(t, errors.Is(token.Error, tc.err), "error %v", token.Error)
		})
	}
}

// TestParsedTokensGracePeriod checks tokens parsed and validated by LoadTokenFromValue.
func TestParsedTokensGracePeriod(t *testing.T) {
	useTestKey(t)
	// expiredJWT has exp and lexp 1337.
	const licenseExpires = 1337
	withinGrace := time.Unix(licenseExpires+1, 0)
	afterGrace := time.Unix(licenseExpires, 0).Add(GracePeriod)

	tests := []struct {
		name     string
		token    string
		pluginID string
		appURL   string
		now      time.Time
		status   TokenStatus
	}{
		{"expired within grace period", expiredJWT, testPluginID, appURL, withinGrace, ExpiredInGracePeriod},
		{"expired after grace period", expiredJWT, testPluginID, appURL, afterGrace, Expired},
		{"other plugin", expiredJWT, "other-plugin", appURL, withinGrace, Invalid},
		{"other app URL", expiredJWT, testPluginID, "http://other.example.com", withinGrace, Invalid},
		{"tampered signature", expiredJWT[:len(expiredJWT)-4] + "AAAA", testPluginID, appURL, withinGrace, Invalid},
		{"malformed", "not-a-token", testPluginID, appURL, withinGrace, Invalid},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			fixedTestTime(t, tc.now)
			token := LoadTokenFromValue(tc.token, tc.appURL, "", tc.pluginID)
			require.Equalf(t, tc.status, token.Status, "error %v", token.Error)
		})
	}
}

// Parse unmarshals the verified payload into the token, so a claim must not be able to set its status.
func TestTokenStatusIsNotAClaim(t *testing.T) {
	var token LicenseToken
	require.NoError(t, json.Unmarshal([]byte(`{"status": 1, "Status": 1}`), &token))
	require.Equal(t, NotLoaded, token.Status)
}
