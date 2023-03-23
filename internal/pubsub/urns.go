package pubsub

import (
	"fmt"
)

func newURN(kind, id string) string {
	return fmt.Sprintf("urn:infratographer:%s:%s", kind, id)
}

// NewTenantURN creates a new tenant URN
func NewTenantURN(id string) string {
	return newURN("tenants", id)
}
