package api

import "errors"

var (
	// ErrInvalidID is returned when a ID is invalid
	ErrInvalidID = errors.New("invalid ID")

	// ErrIDNotFound is returned when a ID is not found in the path
	ErrIDNotFound = errors.New("ID not found in path")

	// ErrTenantNameMissing is returned when the Tenant Name is not defined.
	ErrTenantNameMissing = errors.New("tenant name is missing")
)
