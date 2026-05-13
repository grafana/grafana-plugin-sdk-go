package licensing

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testPluginID  = "grafana-marketplacetest-datasource"
	testProductID = "marketplace-" + testPluginID
	appURL        = "http://grafana.mycompany.com"
)

// Both the private key and the publicKey below have been generated using:
// cd grafana-catalog-team/scripts/marketplace/genjwk
// go run main.go -kid tests

/*
// Private key used for these tests encoded in PEM PKCS#8.
var privateKey = `
-----BEGIN RSA PRIVATE KEY-----
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
-----END RSA PRIVATE KEY-----
`
*/

// Public key of the private key above, returned by running the genjwk script above (priv-key.pub.json)
var publicKey = `{"kty":"RSA","kid":"tests","alg":"RS512","n":"nyfvbGABIlwfGVxvS496I-Stp6RwPorLZ95lrv5VmxZD7yS2HeEd0wy_3E8rOC5kcHU3vAzpf2McDLNsQ1wlDSaBINSTirn3a9lxPASFII40PlzIMSNHXVSSphnz1UiMiZ3m-MiApC5S-zn4c6dz5ObHQ8Lp_8DLcWlYliTTiKwdnuRXN2mBQSKTMc7LlJK2F-FfVpzWNoAZWEJh-ZPp1rsN0d49pko7t5Fiepob_SbqaDonk3yuIhMLIp5l4G8LceiQga3KCsHo6O2yHB8WYloV95r1FbNr016VTm5JNuFgvFUWh2X_dOs-xvmtLvqcPIJHrZwW1vbrnZ9VXwWJj25qiYNaIWqvinFljeJ-WMFzS5n6EPvRTU5RcgWVAPBwfUEpyCaCdyvzcIBfXd2PDUzj-Ev-UOOL1Ewgg1j0EgqUP-TtwvTpUn6Zsz0XxGiDngogAP_0LJELHYgNUt0GtWogun9Keog5WPVbXm1GDeyfiGYt4WpjCf5-eco3YXOokCf5VgYhc_ZXhCbWQb1ZKyX8MVecqg2kEP2Ec92mC9Wx3DgSBEPzHs7gXhT1QeG5W58h7sNjRRExxx2szIQK41BenaYnUNdnI8ggx7b70FWlgXAEyfylycokIpnJpumL-BmkgcD_LNn8env1ecCjXhIs4yKdPJDM3AcZZQZlHPc","e":"AQAB"}`

func TestLoadTokenFromFile(t *testing.T) {
	useTestKey(t)

	tests := []struct {
		name           string
		tokenPath      string
		status         TokenStatus
		expErr         error
		expErrContains string
		token          *LicenseToken
	}{
		{
			name:           "With the wrong file path",
			tokenPath:      "no/such_license_file.jwt",
			status:         NotFound,
			expErr:         errFileNotFound,
			expErrContains: "no/such_license_file.jwt",
		},
		{
			name: "With a valid but expired token",
			// generated with:
			// cd grafana-catalog-team/scripts/marketplace/genjwk
			// go run main.go -company mycompany -priv-key ../genjwk/priv-key.json -products marketplace-grafana-marketplacetest-datasource -subject http://grafana.mycompany.com -expiry 1337
			tokenPath:      "./test-licenses/expired/license.jwt",
			status:         Expired,
			expErr:         errLicenseExpired,
			expErrContains: time.Unix(1577854800, 0).String(),
			token:          createValidToken(t),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token := LoadTokenFromFile(tc.tokenPath, appURL, "", testPluginID)

			require.Equalf(t, tc.status, token.Status, "status should match expected. token error: %q", token.Error)

			if tc.expErr != nil {
				require.Error(t, token.Error)
				require.ErrorIs(t, token.Error, tc.expErr)
				return
			}

			require.NoError(t, token.Error)
			require.Equal(t, tc.token, token)
		})
	}
}

func TestLoadValidationKeys(t *testing.T) {
	fixedTestTime(t, time.Date(2026, 06, 13, 0, 0, 0, 0, time.UTC))

	useTestKey(t)

	// expiry=2099-12-31 23:59:59, lid=10500
	// Generated using:
	// cd grafana-catalog-team/scripts/marketplace/genjwt
	// go run main.go -company mycompany -priv-key ../genjwk/child.json -products marketplace-grafana-marketplacetest-datasource -lid 10500 -subject http://grafana.mycompany.com -expiry 4102444799
	jwt := `eyJhbGciOiJSUzUxMiIsImtpZCI6ImNoaWxkIiwidHlwIjoiSldUIn0.eyJqdGkiOiIxOTA5Mjc5MjkwIiwiaXNzIjoiaHR0cHM6Ly9ncmFmYW5hLmNvbSIsInN1YiI6Imh0dHA6Ly9ncmFmYW5hLm15Y29tcGFueS5jb20iLCJpYXQiOjE3Nzg2ODcyMDUsImV4cCI6NDEwMjQ0NDc5OSwibmJmIjoxNzc4Njg3MjA1LCJsZXhwIjo0MTAyNDQ0Nzk5LCJsaWQiOiIxMDUwMCIsInByb2QiOlsibWFya2V0cGxhY2UtZ3JhZmFuYS1tYXJrZXRwbGFjZXRlc3QtZGF0YXNvdXJjZSJdLCJwcmljaW5nX21vZGVsIjoidXNhZ2UiLCJyZXZlbnVlX3NoYXJlX3BjdCI6NzAsImNvbXBhbnkiOiJteWNvbXBhbnkifQ.PSnttpyemQCk542B2xoYGGsrMtzuqJ9BGvL1YBE6WPTkcUSN2UkoCK7SJjnYibNMNnpTDuOSTZFeNj3hhuSZ3RZZK7fkk99yh4w_1rDDrUjciQxK5sLLsnVqrlO_3oh_0y8JL1ubRCxPG0bB4oKMwcu0oRhfbEfhuSmWSgPvfw5PEUYPDnGPy5eHjalcQE9Fv5ROQewdsl_pDNkpCZmvVI_oRrRz57v5CQAZHq5LUms1n_PK893hRksTfZIngv8LFhwa-txXGGfHWbZFL-ApflUkil3X2bUcUVtrDFVQbpLq1LNaDYucl-AOlLebE_bxmnbkOmdH3bp-KMe1evYZCKANSSUu5OWkYRhNvMLH65STREtI3TDDrcx3ho0av_fp3tW4LewBLzKBNQddDy8xwvwxwONbdrYP-I64rGLuKWg1m49u9EeCVKviCFVi4TrBrNJFQsAAoh9q2Z8lpEecoYBVd6RP8RYY7KCNN-sikQcOqE3GMGIqSFc4yFSw5w8tg-3qzbwesV5lC9MBz6V3rkVNfYnq_wgIl7voDApKymzy17z2xIz3hpKldOxLO9MVYXKa9x45jXLqJH0c_c_WLQarHQSb3tHOtfotplWjB2JkeTzhQvd1LCyC1Z-w52QtWE2HegWHhvaQQV2Tvzchlh1ZelWPxfTnUYn7_bJ3Olo`

	// Generated using:
	// cd grafana-catalog-team/scripts/marketplace/genjwk
	// go run main.go -kid child -out child
	// cd ../genjwks
	// go run main.go -root-priv-key ../genjwk/priv-key.json ../genjwk/child.pub.json
	//
	// (requires grafana-catalog-team/scripts/marketplace/genjwk/priv-key.json to be in place with the correct private key,
	// which is the content of the "privateKey" variable at the top of this file)
	validJWKs := `eyJhbGciOiJSUzUxMiIsImtpZCI6InRlc3RzIiwidHlwIjoiSldUIn0.eyJrZXlzIjpbeyJrdHkiOiJSU0EiLCJraWQiOiJjaGlsZCIsImFsZyI6IlJTNTEyIiwibiI6IndZQThNVjZLbTlmODZtNEZ1UnFTazRTLVdQdExaZTI4Z3FqSmhCOWx2eGVTanBuRTEyRzJOU2NTSElKby1LVFB4MVJWSjJ5dXZpYU9ybmNUdzhkaUU2NExjNHM1RWgtLXRYZTJNSnh4WFozbUQ5SnBYQUhnOWlpaUdSNlJOakxDWWR1cU4tbTR2Q0xMTm4zNXJ5SV9rOUlRb3hEbE55cFFoNzBWZklYdkRpZXo1V2pJTFlicjlzdGxvNHM4eFdxZTRaUHktdHk5b3JrSUxucTh2ajdVYlBMSW91M1RZVGpnOTUtbTZJdktFNnQ4VGc0ZkFQbVgzZWZTcVlHWENWeEdEWjUzQ0lxVmhnZG90cTRjZndjRUpMbHdvdlh1bk13Z0VGMERiMUhmdVlYQmFlLUs5NW5pemk2MDY0OUNzcy1GTExsNGhmN2s1UTYzSWxRTkdtQzFZaEtTZ3VXT3dkNm9pVmZBTDdUT3pXTlNzcjZiMnJJcldpZUMxUWo1X1ZXMXlMejVoTVN2VGMzTFRBOHlxd1J0VnNPSE93V0wwMF9UcFJlNXh6UmNPM0MwSjVUdDNMNGtxN0pWT2tiUi0tMmNwNXdtcURIU2xkUEI5WGFfRVllWm45aG50SVFvWFdrb1pkbnhkLU1vcjdLaGdyWGtZTFcyeHE4MV9ZZkJEYmJVc0pDWGtVMUdWYXUyLTdQTWlyRm03VXE4V1d3Q0trRXBUQ1Nrd28tQUd0bG4wc08xOTRXOEtVZ1Nlb05HZXRiMTNpa2JvZUNLRE5mbUR1aFR6LWJIU1BPdzBWZHg5OWFIbEUwRnNpTl95RWFUbU4wOEdZNWY0MHpMdUxPMlYzUU5sdWg4N1duZkdMR3Y3ZV8xYVE3bFZWaFBDRTdFcEZtWklkT0dKaUhhN0s4IiwiZSI6IkFRQUIifV19.fI8Oc76MQ9IXiiK_Q6MozHtSNRMUSIri2uc5Md_iSeQPoD4iJK0wIqwqMaYlyhKNs41F6JnjuyjNjmt4nzKTMzlvbJSXxhGXP0cGStcG9joHDyg1R05HHFAuuc-lgKQwfbjqRt5PdHryOCpuyntAyCfgJFEQElsu9mfE1aisv5H3MJaUihvkVhjtRV_KHvBdYCu3T-w4ntU8cBpyiKu8h12gmMK-BbGRpWdpXBm7Iz8LL1CPVas71_1aKUAGtzGVc76c3mvolGzFF5ygVvZmptDK2UD3uDeALcjXTr0s3Lfmhsqpz1WMtSJBTEBTfXuNregsVS-I4pFJxyqo9XMnEG1p_hJondmSaY14VeHUJH6icu0DWRj5cn3xellqU36VhzOqZweNmRi9Yo444_3hkpEwp_gbcJqDOZpM0uz6jjdhnjyeAsKpHHZMLN3WJCkTuIhefd6cJMi5j6ir6-NJyhStjE0ZzjxmKD5OL42wrSdfoT71N3gt91BDkFs0w2J39304z70u8-7-av_Fpzz6aUvRi0EuTptXI2uEXbIBxGqUfYU6cSHgzFg8FaFXrb-k1pvudL8Eapom6KcbaNwDF84OVuQiijZZI0piG8RDxBDDq15VnCsk2c90mTToWL1cXXB-WD_EfqF4c8TXKQx6FbPWhjuI5Ams0I68AbnGGYw`

	tests := []struct {
		name        string
		jwks        string
		bLicenses   map[string]struct{}
		expValidity bool
	}{
		{
			name:        "empty",
			jwks:        "",
			bLicenses:   map[string]struct{}{},
			expValidity: false,
		},
		{
			name:        "valid jwks and valid jwt",
			jwks:        validJWKs,
			bLicenses:   map[string]struct{}{"12345": {}},
			expValidity: true,
		},
		{
			name: "valid jwks and revoked jwt",
			jwks: validJWKs,
			// `10500` must match the `lid` field in the decoded `jwt` above
			bLicenses:   map[string]struct{}{"10500": {}},
			expValidity: false,
		},
		{
			name:        "invalid jwks",
			jwks:        jwt,
			bLicenses:   map[string]struct{}{"12345": {}},
			expValidity: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.bLicenses) != 0 {
				useBlockedLicenseIds(t, tc.bLicenses)
			}
			token := &LicenseToken{}
			status := token.Parse(jwt, appURL, tc.jwks, testPluginID)
			assert.Equalf(t, tc.expValidity, status, "token validity should match expected. error: %q", token.Error)
		})
	}
}

func createValidToken(t *testing.T) *LicenseToken {
	t.Helper()
	raw, err := os.ReadFile("./test-licenses/expired/license.jwt")
	require.NoError(t, err)
	return &LicenseToken{
		Raw:             string(raw),
		Status:          Valid,
		Error:           nil,
		Id:              "14",
		Issuer:          "http://raintank-dev:4000",
		Subject:         appURL,
		Issued:          1539191907,
		Expires:         1577854800,
		LicenseIssued:   1539191759,
		LicenseExpires:  1577854800,
		LicenseId:       "5",
		IncludedAdmins:  -1,
		IncludedViewers: -1,
		IncludedUsers:   -1,
		// LicenseExpiresWarnDays: 0,
		Products: []string{testProductID},
		Company:  "raintank",
	}
}

func useTestKey(tb testing.TB) {
	tb.Helper()
	embeddedKeys["tests"] = publicKey
	tb.Cleanup(func() {
		delete(embeddedKeys, "tests")
	})
}
