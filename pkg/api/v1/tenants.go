package api

import (
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"go.infratographer.com/tenant-api/internal/models"
	"go.infratographer.com/tenant-api/internal/pubsub"
	"go.infratographer.com/tenant-api/pkg/jwtauth"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func (r *Router) tenantCreate(c echo.Context) error {
	tenantID, err := parseUUID(c, "id")
	if err != nil && !errors.Is(err, ErrUUIDNotFound) {
		r.logger.Error("invalid tenant uuid", zap.Error(err))

		return v1BadRequestResponse(c, err)
	}

	traceOpts := []trace.SpanStartOption{}
	if tenantID != "" {
		traceOpts = append(traceOpts, trace.WithAttributes(attribute.String("tenant-id", tenantID)))
	}

	ctx, span := tracer.Start(c.Request().Context(), "tenantCreate", traceOpts...)
	defer span.End()

	createRequest := new(createTenantRequest)

	if err := c.Bind(createRequest); err != nil {
		r.logger.Error("failed to bind tenant create request", zap.Error(err))

		return v1BadRequestResponse(c, err)
	}

	if err := createRequest.validate(); err != nil {
		r.logger.Error("invalid create request", zap.Error(err))

		return v1BadRequestResponse(c, err)
	}

	t := &models.Tenant{
		Name: createRequest.Name,
	}

	var additionalURNs []string

	if tenantID != "" {
		t.ParentTenantID = null.StringFrom(tenantID)
		additionalURNs = append(additionalURNs, pubsub.NewTenantURN(tenantID))
	}

	if err := t.Insert(ctx, r.db, boil.Infer()); err != nil {
		r.logger.Error("error inserting tenant", zap.Error(err))

		return v1InternalServerErrorResponse(c, err)
	}

	actor := jwtauth.Actor(c)

	msg, err := pubsub.NewTenantMessage(
		actor,
		pubsub.NewTenantURN(t.ID),
		additionalURNs...,
	)
	if err != nil {
		// TODO: add status to reconcile and requeue this
		r.logger.Error("failed to create tenant message", zap.Error(err))
	}

	if err := r.pubsub.PublishCreate(ctx, "tenants", "global", msg); err != nil {
		// TODO: add status to reconcile and requeue this
		r.logger.Error("failed to publish tenant message", zap.Error(err))
	}

	return v1TenantCreatedResponse(c, t)
}

func (r *Router) tenantList(c echo.Context) error {
	pagination := parsePagination(c)

	ctx, span := tracer.Start(c.Request().Context(), "tenantList")
	defer span.End()

	var mods []qm.QueryMod

	if tenantID, err := parseUUID(c, "id"); err == nil {
		mods = append(mods, models.TenantWhere.ParentTenantID.EQ(null.StringFrom(tenantID)))
	} else if errors.Is(err, ErrUUIDNotFound) {
		mods = append(mods, models.TenantWhere.ParentTenantID.IsNull())
	} else {
		return v1BadRequestResponse(c, err)
	}

	mods = append(mods, pagination.queryMods()...)

	ts, err := models.Tenants(mods...).All(ctx, r.db)
	if err != nil {
		r.logger.Error("failed to query tenants", zap.Error(err))

		return v1InternalServerErrorResponse(c, err)
	}

	return v1TenantsResponse(c, ts, pagination)
}

func v1Tenant(t *models.Tenant) *tenant {
	return &tenant{
		ID:             t.ID,
		Name:           t.Name,
		ParentTenantID: t.ParentTenantID.Ptr(),
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
		DeletedAt:      t.DeletedAt.Ptr(),
	}
}

func v1TenantSlice(ts []*models.Tenant) tenantSlice {
	tenants := make(tenantSlice, len(ts))

	for i, t := range ts {
		tenants[i] = v1Tenant(t)
	}

	return tenants
}
