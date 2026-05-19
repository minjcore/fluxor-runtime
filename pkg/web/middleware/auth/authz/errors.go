package authz

import "errors"

var (
	// ErrUnauthorized is returned when authorization fails
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when access is forbidden
	ErrForbidden = errors.New("forbidden")

	// ErrInvalidRequest is returned when an authorization request is invalid
	ErrInvalidRequest = errors.New("invalid authorization request")

	// ErrPolicyNotFound is returned when a policy is not found
	ErrPolicyNotFound = errors.New("policy not found")

	// ErrInvalidPolicy is returned when a policy is invalid
	ErrInvalidPolicy = errors.New("invalid policy")
)
