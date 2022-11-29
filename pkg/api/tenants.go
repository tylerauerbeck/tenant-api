package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"go.infratographer.com/identityapi/internal/models"
	"go.infratographer.com/identityapi/pkg/checker"
	"go.infratographer.com/permissionapi/pkg/pubsubx"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	ouOrderDefault    = ""
	ouOrderNewest     = "newest"
	ouOrderOldest     = "oldest"
	ouOrderEditedLast = "edited"
	ouOrderName       = "name"
)

type Actor struct {
	URN   string
	Token string
}

var ErrUnauthenticatedRequest = errors.New("unauthenticated request")

func getActor(c echo.Context) (*Actor, error) {
	authHeader := c.Request().Header.Get("Infratographer-Auth")

	if authHeader == "" {
		return nil, ErrUnauthenticatedRequest
	}

	return &Actor{URN: authHeader}, nil
}

func (r *Router) tenantCreate(c echo.Context) error {
	tenantID := c.Param("id")

	traceOpts := []trace.SpanStartOption{}
	if tenantID != "" {
		traceOpts = append(traceOpts, trace.WithAttributes(attribute.String("tenant-id", tenantID)))
	}

	ctx, span := tracer.Start(c.Request().Context(), "tenantCreate", traceOpts...)
	defer span.End()

	input := struct {
		ID             uuid.NullUUID `form:"id" json:"id"`
		Name           string        `form:"name" json:"name" binding:"required"`
		ParentTenantID uuid.NullUUID `query:"id"`
	}{}

	// if err := c.ShouldBindJSON(&input); err != nil {
	// 	return err
	// }

	if err := c.Bind(&input); err != nil {
		return err
	}

	actor, err := getActor(c)
	if err != nil {
		return err
	}

	if tenantID == "" {
		if ok, err := r.permsClient.ActorHasGlobalScope(ctx, actor.URN, string(checker.GlobalScopeRootTenantCreate)); !ok {
			if err != nil {
				return err
			}

			return err
		}
	} else {
		if ok, err := r.permsClient.ActorHasScope(ctx, actor.URN, checker.ScopeTenantCreate, fmt.Sprintf("urn:infratographer:tenant:%s", tenantID)); !ok {
			if err != nil {
				return err
			}

			return err
		}
	}

	t := &models.Tenant{Name: input.Name}

	if input.ID.Valid {
		t.ID = input.ID.UUID.String()
	}

	if tenantID != "" {
		t.ParentTenantID = null.StringFrom(tenantID)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	err = t.Insert(ctx, r.db, boil.Infer())
	if err != nil {
		return err
	}

	// err = r.permsClient.CreateResource(ctx, actor.URN, fmt.Sprintf("urn:infratographer:tenant:%s", t.ID), v1Tenant(t))
	// if err != nil {
	// 	return err
	// }

	msg := &pubsubx.Message{
		SubjectURN: "urn:infratographer:tenant:" + t.ID,
		EventType:  "tenant.added",
		ActorURN:   actor.URN,
		Source:     "identityapi",
		Timestamp:  time.Now(),
		SubjectFields: map[string]string{
			"id":         t.ID,
			"name":       t.Name,
			"created_at": t.CreatedAt.Format(time.RFC3339),
			"updated_at": t.UpdatedAt.Format(time.RFC3339),
		},
	}

	if t.ParentTenantID.Valid {
		msg.AdditionalSubjectURNs = append(msg.AdditionalSubjectURNs, "urn:infratographer:tenant:"+t.ParentTenantID.String)
		msg.SubjectFields["parent_tenant_id"] = t.ParentTenantID.String
	}

	err = pubsubx.HackySendMsg(ctx, "com.infratographer.events.tenant.added", msg)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, t)
}

func (r *Router) tenantList(c echo.Context) error {
	ctx, span := tracer.Start(c.Request().Context(), "tenantList")
	defer span.End()

	actor, err := getActor(c)
	if err != nil {
		return err
	}

	// authIDs, err := r.permsClient.ResourcesAvailable("tenant", checker.ScopeTenantMember, "user")
	authIDs, err := r.permsClient.ResourcesAvailable(ctx, actor.URN, "urn:infratographer:tenant", "direct_member")
	if err != nil {
		return err
	}

	mods := []qm.QueryMod{models.TenantWhere.ID.IN(authIDs)}

	// mods = append(mods, RequestQueryMods(c)...)

	// switch GetOrder(c) {
	// case ouOrderDefault, ouOrderNewest:
	// 	mods = append(mods, qm.OrderBy(models.TenantTableColumns.CreatedAt+" DESC"))
	// case ouOrderOldest:
	// 	mods = append(mods, qm.OrderBy(models.TenantTableColumns.CreatedAt+" ASC"))
	// case ouOrderEditedLast:
	// 	mods = append(mods, qm.OrderBy(models.TenantTableColumns.UpdatedAt+" DESC"))
	// case ouOrderName:
	// 	mods = append(mods, qm.OrderBy(models.TenantTableColumns.Name+" DESC"))
	// default:
	// 	mods = append(mods, qm.OrderBy(models.TenantTableColumns.CreatedAt+" DESC"))
	// }

	ts, err := models.Tenants(mods...).All(ctx, r.db)
	if err != nil {
		// c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "database": "goes boom"})
		return err
	}

	// listResponse(c, srvs, pd)

	return c.JSON(http.StatusOK, v1TenantSlice(ts))
}

func v1Tenant(t *models.Tenant) any {
	return struct {
		ID             string     `json:"id"`
		Name           string     `json:"name"`
		ParentTenantID *string    `json:"parent_tenant_id,omitempty"`
		CreatedAt      time.Time  `json:"created_at"`
		UpdatedAt      time.Time  `json:"updated_at"`
		DeletedAt      *time.Time `json:"deleted_at,omitempty"`
	}{
		ID:             t.ID,
		Name:           t.Name,
		ParentTenantID: t.ParentTenantID.Ptr(),
		CreatedAt:      t.CreatedAt,
		UpdatedAt:      t.UpdatedAt,
		DeletedAt:      t.DeletedAt.Ptr(),
	}
}

func v1TenantSlice(ts []*models.Tenant) []any {
	r := []any{}

	for _, t := range ts {
		r = append(r, v1Tenant(t))
	}

	return r
}
