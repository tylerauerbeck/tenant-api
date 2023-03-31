package api

import "errors"

var (
	// ErrInvalidUUID is returned when a UUID is invalid
	ErrInvalidUUID = errors.New("invalid UUID")

	// ErrUUIDNotFound is returned when a UUID is not found in the path
	ErrUUIDNotFound = errors.New("UUID not found in path")

	// ErrTenantNameMissing is returned when the Tenant Name is not defined.
	ErrTenantNameMissing = errors.New("tenant name is missing")
)
