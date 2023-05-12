package api

import (
	"time"

	"go.infratographer.com/x/gidx"
)

type tenantSlice []*tenant

type tenant struct {
	ID             gidx.PrefixedID  `json:"id"`
	Name           string           `json:"name"`
	ParentTenantID *gidx.PrefixedID `json:"parent_tenant_id,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
	DeletedAt      *time.Time       `json:"deleted_at,omitempty"`
}
