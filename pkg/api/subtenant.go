package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"go.infratographer.com/identityapi/internal/models"
)

func (r *Router) subTenantList(c *gin.Context) {
	tenantID := c.Param("tenant-id")

	ctx, span := tracer.Start(c.Request.Context(), "subTenantList", trace.WithAttributes(attribute.String("tenant-id", tenantID)))
	defer span.End()

	_, err := uuid.Parse(tenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "tenant id is not a valid UUID"})
	}

	// if !authChecker.ActorHasScope("nicole", checker.ScopeTenantMember, "tenant", tenantID) {
	// 	c.JSON(http.StatusNotFound, gin.H{"error": "tenant not found or you don't have access to it"})
	//  return
	// }

	mods := []qm.QueryMod{models.TenantWhere.ParentTenantID.EQ(null.StringFrom(tenantID))}

	mods = append(mods, RequestQueryMods(c)...)

	switch GetOrder(c) {
	case ouOrderDefault, ouOrderNewest:
		mods = append(mods, qm.OrderBy(models.TenantTableColumns.CreatedAt+" DESC"))
	case ouOrderOldest:
		mods = append(mods, qm.OrderBy(models.TenantTableColumns.CreatedAt+" ASC"))
	case ouOrderEditedLast:
		mods = append(mods, qm.OrderBy(models.TenantTableColumns.UpdatedAt+" DESC"))
	case ouOrderName:
		mods = append(mods, qm.OrderBy(models.TenantTableColumns.Name+" DESC"))
	default:
		mods = append(mods, qm.OrderBy(models.TenantTableColumns.CreatedAt+" DESC"))
	}

	t, err := models.Tenants(mods...).All(ctx, r.db)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "database": "goes boom"})
		return
	}

	// listResponse(c, srvs, pd)
	c.JSON(http.StatusOK, t)
}
