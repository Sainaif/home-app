package utils

import (
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"net/url"

	"github.com/pquerna/otp/totp"
)

// GenerateTOTPSecret generates a new TOTP secret
func GenerateTOTPSecret() (string, error) {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	return base32.StdEncoding.EncodeToString(secret), nil
}

// GenerateTOTPURL generates a TOTP provisioning URL for QR code
func GenerateTOTPURL(secret, email, issuer string) string {
	params := url.Values{}
	params.Set("secret", secret)
	params.Set("issuer", issuer)
	params.Set("algorithm", "SHA1")
	params.Set("digits", "6")
	params.Set("period", "30")

	u := url.URL{
		Scheme:   "otpauth",
		Host:     "totp",
		Path:     fmt.Sprintf("/%s:%s", issuer, email),
		RawQuery: params.Encode(),
	}

	return u.String()
}

// ValidateTOTP validates a TOTP code against a secret
func ValidateTOTP(code, secret string) bool {
	return totp.Validate(code, secret)
}
