package pubsub

import (
	"go.infratographer.com/x/gidx"
	"go.infratographer.com/x/pubsubx"
)

// NewTenantMessage creates a new tenant event message
func NewTenantMessage(actorID, tenantID gidx.PrefixedID, additionalSubjectIDs ...gidx.PrefixedID) (*pubsubx.ChangeMessage, error) {
	return newMessage(actorID, tenantID, additionalSubjectIDs...), nil
}

// UpdateTenantMessage creates a updated tenant event message
func UpdateTenantMessage(actorID, tenantID gidx.PrefixedID, additionalSubjectIDs ...gidx.PrefixedID) (*pubsubx.ChangeMessage, error) {
	return newMessage(actorID, tenantID, additionalSubjectIDs...), nil
}

// DeleteTenantMessage creates a delete tenant event message
func DeleteTenantMessage(actorID, tenantID gidx.PrefixedID, additionalSubjectIDs ...gidx.PrefixedID) (*pubsubx.ChangeMessage, error) {
	return newMessage(actorID, tenantID, additionalSubjectIDs...), nil
}
