package api

import (
	"database/sql"
	"errors"

	"github.com/labstack/echo/v4"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"go.infratographer.com/tenant-api/internal/models"
	"go.infratographer.com/tenant-api/internal/pubsub"
	"go.infratographer.com/x/echojwtx"
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

	actor := echojwtx.Actor(c)

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

func (r *Router) tenantGet(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "tenantGet")
	defer span.End()

	var mods []qm.QueryMod

	tenantID, err := parseUUID(c, "id")
	if err != nil {
		return v1BadRequestResponse(c, err)
	}

	mods = append(mods, models.TenantWhere.ID.EQ(tenantID))

	t, err := models.Tenants(mods...).One(ctx, r.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return v1TenantNotFoundResponse(c, err)
		}

		r.logger.Error("failed to query tenants", zap.Error(err))

		return v1InternalServerErrorResponse(c, err)
	}

	return v1TenantGetResponse(c, t)
}

func (r *Router) tenantUpdate(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "tenantUpdate")
	defer span.End()

	var mods []qm.QueryMod

	tenantID, err := parseUUID(c, "id")
	if err != nil {
		return v1BadRequestResponse(c, err)
	}

	payload := new(updateTenantRequest)

	if err := c.Bind(&payload); err != nil {
		r.logger.Error("failed to bind update tenant request", zap.Error(err))

		return v1BadRequestResponse(c, err)
	}

	if err := payload.validate(); err != nil {
		r.logger.Error("invalid update tenant request", zap.Error(err))

		return v1BadRequestResponse(c, err)
	}

	mods = append(mods, models.TenantWhere.ID.EQ(tenantID))

	t, err := models.Tenants(mods...).One(ctx, r.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return v1TenantNotFoundResponse(c, err)
		}

		r.logger.Error("failed to query tenants", zap.Error(err))

		return v1InternalServerErrorResponse(c, err)
	}

	if payload.Name != nil {
		t.Name = *payload.Name
	}

	if _, err := t.Update(ctx, r.db, boil.Infer()); err != nil {
		r.logger.Error("failed to update tenant", zap.Error(err))

		return v1InternalServerErrorResponse(c, err)
	}

	actor := echojwtx.Actor(c)

	msg, err := pubsub.UpdateTenantMessage(
		actor,
		pubsub.NewTenantURN(t.ID),
	)
	if err != nil {
		// TODO: add status to reconcile and requeue this
		r.logger.Error("failed to create, update tenant message", zap.Error(err))
	}

	if err := r.pubsub.PublishUpdate(ctx, "tenants", "global", msg); err != nil {
		// TODO: add status to reconcile and requeue this
		r.logger.Error("failed to publish, update tenant message", zap.Error(err))
	}

	return v1TenantGetResponse(c, t)
}

func (r *Router) tenantDelete(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "tenantDelete")
	defer span.End()

	var mods []qm.QueryMod

	tenantID, err := parseUUID(c, "id")
	if err != nil {
		return v1BadRequestResponse(c, err)
	}

	mods = append(mods, models.TenantWhere.ID.EQ(tenantID))

	t, err := models.Tenants(mods...).One(ctx, r.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return v1TenantNotFoundResponse(c, err)
		}

		r.logger.Error("failed to query tenants", zap.Error(err))

		return v1InternalServerErrorResponse(c, err)
	}

	if _, err := t.Delete(ctx, r.db, false); err != nil {
		r.logger.Error("failed to delete tenant", zap.Error(err))

		return err
	}

	actor := echojwtx.Actor(c)

	msg, err := pubsub.DeleteTenantMessage(
		actor,
		pubsub.NewTenantURN(t.ID),
	)
	if err != nil {
		// TODO: add status to reconcile and requeue this
		r.logger.Error("failed to create, delete tenant message", zap.Error(err))
	}

	if err := r.pubsub.PublishDelete(ctx, "tenants", "global", msg); err != nil {
		// TODO: add status to reconcile and requeue this
		r.logger.Error("failed to publish, delete tenant message", zap.Error(err))
	}

	return nil
}

const (
	parentsQuery = `
		WITH RECURSIVE get_parents AS (
			SELECT id, name, parent_tenant_id, created_at, updated_at, deleted_at
			FROM tenants
			WHERE
				id = $1
				AND deleted_at IS NULL

			UNION (
				SELECT t.id, t.name, t.parent_tenant_id, t.created_at, t.updated_at, t.deleted_at
				FROM tenants t
				INNER JOIN get_parents gp ON t.id = gp.parent_tenant_id
				WHERE t.deleted_at IS NULL
				ORDER BY created_at
			)
		)
		SELECT id, name, parent_tenant_id, created_at, updated_at, deleted_at
		FROM get_parents
	`
	parentsUntilQuery = `
		WITH RECURSIVE get_parents AS (
			SELECT id, name, parent_tenant_id, created_at, updated_at, deleted_at
			FROM tenants
			WHERE
				id = $1
				AND deleted_at IS NULL

			UNION (
				SELECT t.id, t.name, t.parent_tenant_id, t.created_at, t.updated_at, t.deleted_at
				FROM tenants t
				INNER JOIN get_parents gp ON t.id = gp.parent_tenant_id
				WHERE
					gp.id != $2
					AND t.deleted_at IS NULL
				ORDER BY created_at
			)
		)
		SELECT id, name, parent_tenant_id, created_at, updated_at, deleted_at
		FROM get_parents
	`
)

func (r *Router) tenantParentsList(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "tenantParentsList")
	defer span.End()

	pagination := parsePagination(c)

	tenantID, err := parseUUID(c, "id")
	if err != nil {
		return v1BadRequestResponse(c, err)
	}

	parentID, err := parseUUID(c, "parent_id")
	if err != nil && !errors.Is(err, ErrUUIDNotFound) {
		return v1BadRequestResponse(c, err)
	}

	var rows *sql.Rows

	if parentID == "" {
		rows, err = r.db.QueryContext(ctx, parentsQuery, tenantID)
	} else {
		rows, err = r.db.QueryContext(ctx, parentsUntilQuery, tenantID, parentID)
	}

	if err != nil {
		r.logger.Error("failed to query tenant parents", zap.Error(err))

		return v1InternalServerErrorResponse(c, err)
	}

	var tenants []*models.Tenant

	for rows.Next() {
		tenant := new(models.Tenant)

		err = rows.Scan(
			&tenant.ID,
			&tenant.Name,
			&tenant.ParentTenantID,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
			&tenant.DeletedAt,
		)

		if err != nil {
			return v1InternalServerErrorResponse(c, err)
		}

		tenants = append(tenants, tenant)
	}

	if len(tenants) == 0 {
		return v1TenantNotFoundResponse(c, nil)
	}

	if pagination.getPageOffset()+1 >= len(tenants) {
		return v1TenantsResponse(c, nil, pagination)
	}

	limit := pagination.getPageOffset() + 1 + pagination.limitUsed()
	if limit > len(tenants) {
		limit = len(tenants)
	}

	return v1TenantsResponse(c, tenants[pagination.getPageOffset()+1:limit], pagination)
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
