package auth

import (
	"errors"
	"net/http"
	"strings"
)

const (
	GoogleAuthCodeInvalid        = "INVALID_CODE"
	GoogleAuthEmailNotVerified   = "EMAIL_NOT_VERIFIED"
	GoogleAuthTokenVerifyFailed  = "TOKEN_VERIFY_FAILED"
	GoogleAuthAccountExistsEmail = "ACCOUNT_EXISTS_EMAIL"
	GoogleAuthUpstreamAPIError   = "GOOGLE_API_ERROR"
	GoogleAuthInvalidRedirectURI = "INVALID_REDIRECT_URI"
)

func GetAPIKey(headers http.Header) (string, error) {
	val := headers.Get("Authorization")
	if val == "" {
		return "", errors.New("no authentication info found")
	}

	vals := strings.Split(val, "")
	if len(vals) != 2 {
		return "", errors.New("malformed auth headers")
	}
	if vals[0] != "ApiKey" {
		return "", errors.New("malformed first part of auth header")
	}
	return vals[1], nil
}
