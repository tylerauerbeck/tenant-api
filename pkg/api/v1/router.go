package api

import (
	"database/sql"
	"net/http"

	"github.com/labstack/echo/v4"
	"go.infratographer.com/tenant-api/pkg/jwtauth"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

const (
	apiVersion = "v1"
)

var tracer = otel.Tracer("go.infratographer.com/tenant-api/pkg/api/v1")

// Router provides a router for the API
type Router struct {
	db     *sql.DB
	logger *zap.Logger
	auth   *jwtauth.Auth
}

func NewRouter(db *sql.DB, l *zap.Logger, auth *jwtauth.Auth) *Router {
	return &Router{
		db:     db,
		logger: l.Named("api"),
		auth:   auth,
	}
}

func errorHandler(err error, c echo.Context) {
	c.Echo().DefaultHTTPErrorHandler(err, c)
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
func (r *Router) Routes(e *echo.Echo) {
	// authenticate a request, not included the v1 group since this has custom
	// authentication as it's accepting external auth
	e.HideBanner = true

	e.HTTPErrorHandler = errorHandler

	e.Use(defaultRequestType)

	// Health endpoints
	e.GET("/healthz", r.livenessCheck)
	e.GET("/readyz", r.readinessCheck)

	v1 := e.Group(apiVersion)
	{
		v1.GET("/", r.apiVersion)

		v1.Use(r.auth.Middleware())

		v1.GET("/tenants", r.tenantList)
		v1.POST("/tenants", r.tenantCreate)

		// v1.GET("/tenants/:id", r.tenantGet)
		// v1.PATCH("/tenants/:id", r.tentantUpdate)
		// v1.DELETE("/tenants/:id", r.tenantDelete)

		v1.GET("/tenants/:id/tenants", r.tenantList)
		v1.POST("/tenants/:id/tenants", r.tenantCreate)
	}
}

// livenessCheck ensures that the server is up and responding
func (r *Router) livenessCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "UP",
	})
}

// readinessCheck ensures that the server is up and that we are able to process
// requests. currently this only checks the database connection.
func (r *Router) readinessCheck(c echo.Context) error {
	ctx := c.Request().Context()

	if err := r.db.PingContext(ctx); err != nil {
		r.logger.Error("readiness check db ping failed", zap.Error(err))

		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status": "DOWN",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status": "UP",
	})
}

// apiVersion responds with the current api version.
func (r *Router) apiVersion(c echo.Context) error {
	return c.JSON(http.StatusOK, echo.Map{
		"version": apiVersion,
	})
}
