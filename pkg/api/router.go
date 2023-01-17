package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
	permissions "go.infratographer.com/permissions-api/pkg/client/v1"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
)

var tracer = otel.Tracer("go.infratographer.com/identity-api/internal/api")

// Router provides a router for the API
type Router struct {
	db          *sql.DB
	logger      *zap.SugaredLogger
	permsClient *permissions.Client
}

func NewRouter(db *sql.DB, l *zap.SugaredLogger) *Router {
	pc, _ := permissions.New("http://localhost:7602", nil)

	return &Router{
		db:          db,
		logger:      l.Named("api"),
		permsClient: pc,
	}
}

func notYet(c echo.Context) error {
	for k, v := range c.Request().Header {
		// if k == "Authorization" {
		fmt.Printf("Header :: %s: %s\n", k, strings.Join(v, ", "))
		// }
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "endpoint not implemented yet"})
}

func errorHandler(err error, c echo.Context) {
	c.Echo().DefaultHTTPErrorHandler(err, c)
}

func DefaultRequestType(next echo.HandlerFunc) echo.HandlerFunc {
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

	e.GET("api/v1/auth/request", r.authRequest)

	e.Use(DefaultRequestType)

	v1 := e.Group("api/v1")
	{
		v1.GET("/tenants", r.tenantList)
		v1.POST("/tenants", r.tenantCreate)
		v1.GET("/tenants/:id", notYet)
		v1.PATCH("/tenants/:id", notYet)
		v1.DELETE("/tenants/:id", notYet)
		v1.GET("/tenants/:id/tenants", notYet) // r.subTenantList
		v1.POST("/tenants/:id/tenants", r.tenantCreate)
	}
}
