package authn

import "errors"

var (
	// ErrInvalidCredential is returned when a credential is invalid
	ErrInvalidCredential = errors.New("invalid credential")

	// ErrExpiredCredential is returned when a credential has expired
	ErrExpiredCredential = errors.New("credential has expired")

	// ErrMissingCredential is returned when a credential is missing
	ErrMissingCredential = errors.New("credential is missing")

	// ErrUnauthorized is returned when authentication fails
	ErrUnauthorized = errors.New("unauthorized")

	// ErrTokenExpired is returned when a token has expired
	ErrTokenExpired = errors.New("token has expired")

	// ErrTokenInvalid is returned when a token is invalid
	ErrTokenInvalid = errors.New("token is invalid")

	// ErrProviderUnavailable is returned when an authentication provider is unavailable
	ErrProviderUnavailable = errors.New("authentication provider unavailable")
)
