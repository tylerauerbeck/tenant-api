package graphapi_test

import (
	"context"

	"github.com/brianvoe/gofakeit/v6"

	ent "go.infratographer.com/tenant-api/internal/ent/generated"
)

type TenantBuilder struct {
	Name        string
	Description string
	Parent      *ent.Tenant
}

func (b TenantBuilder) MustNew(ctx context.Context) *ent.Tenant {
	if b.Name == "" {
		b.Name = gofakeit.Company()
	}

	if b.Description == "" {
		b.Description = gofakeit.Company()
	}

	input := ent.CreateTenantInput{
		Name:        b.Name,
		Description: &b.Description,
	}

	if b.Parent != nil {
		input.ParentID = &b.Parent.ID
	}

	return testTools.entClient.Tenant.Create().SetInput(input).SaveX(ctx)
}
