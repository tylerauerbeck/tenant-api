// Package schema contains the ent schema definitions for the tenant API.
package schema

const (
	// ApplicationPrefix is the prefix for all application IDs owned by tenant-api
	ApplicationPrefix string = "tnnt"
	// TenantPrefix is the prefix for tenants
	TenantPrefix string = ApplicationPrefix + "ten"
)
