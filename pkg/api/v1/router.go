package api

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.infratographer.com/tenant-api/internal/pubsub"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

const (
	apiVersion = "v1"
)

var tracer = otel.Tracer("go.infratographer.com/tenant-api/pkg/api/v1")

// Router provides a router for the API
type Router struct {
	db         *sql.DB
	logger     *zap.Logger
	pubsub     *pubsub.Client
	middleware []echo.MiddlewareFunc
}

// NewRouter creates a new APIv1 router.
func NewRouter(db *sql.DB, ps *pubsub.Client, options ...RouterOption) *Router {
	router := &Router{
		db:     db,
		logger: zap.NewNop(),
		pubsub: ps,
	}

	for _, opt := range options {
		opt(router)
	}

	return router
}

// Ensures request header Content-Type is set to application/json if not already defined.
func defaultRequestType(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		contentType := c.Request().Header.Get("Content-Type")
		if contentType == "" {
			c.Request().Header.Set("Content-Type", "application/json")
		}

		return next(c)
	}
}

// Routes will add the routes for this API version to a router group
func (r *Router) Routes(e *echo.Group) {
	v1 := e.Group(apiVersion)
	{
		v1.Use(defaultRequestType)
		v1.Use(r.middleware...)

		v1.GET("/", r.apiVersion)

		v1.GET("/tenants", r.tenantList)
		v1.POST("/tenants", r.tenantCreate)

		v1.GET("/tenants/:id", r.tenantGet)
		v1.PATCH("/tenants/:id", r.tenantUpdate)
		v1.DELETE("/tenants/:id", r.tenantDelete)

		v1.GET("/tenants/:id/tenants", r.tenantList)
		v1.POST("/tenants/:id/tenants", r.tenantCreate)

		v1.GET("/tenants/:id/parents", r.tenantParentsList)
		v1.GET("/tenants/:id/parents/:parent_id", r.tenantParentsList)
	}

	_, err := r.pubsub.AddStream()
	if err != nil {
		r.logger.Fatal("failed to add stream", zap.Error(err))
	}
}

// DatabaseCheck ensure the database connection is established.
func (r *Router) DatabaseCheck(ctx context.Context) error {
	if err := r.db.PingContext(ctx); err != nil {
		return err
	}

	return nil
}

// apiVersion responds with the current api version.
func (r *Router) apiVersion(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"version": apiVersion,
	})
}
