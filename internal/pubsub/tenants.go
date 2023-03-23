package pubsub

import (
	"go.infratographer.com/x/pubsubx"
)

// NewTenantMessage creates a new tenant event message
func NewTenantMessage(actorURN string, tenantURN string, additionalSubjectURNs ...string) (*pubsubx.Message, error) {
	return newMessage(actorURN, tenantURN, additionalSubjectURNs...), nil
}
