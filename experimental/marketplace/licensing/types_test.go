package licensing

import (
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
			},
			appURL:         appURL,
			bLicenses:      bLicenses,
			status:         Expired,
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
			},
			appURL:         appURL,
			bLicenses:      bLicenses,
			status:         Expired,
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
			status:         InvalidSubject,
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
			status:    InvalidSubject,
			expErr:    errInvalidAppURL,
		},
		{
			name: "With an blocked licenses ID",
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
