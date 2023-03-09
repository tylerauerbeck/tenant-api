package api

import "time"

type tenantSlice []*tenant

type tenant struct {
	ID             string     `json:"id"`
	Name           string     `json:"name"`
	ParentTenantID *string    `json:"parent_tenant_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	DeletedAt      *time.Time `json:"deleted_at,omitempty"`
}
