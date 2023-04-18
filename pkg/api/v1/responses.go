package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"go.infratographer.com/tenant-api/internal/models"
)

type v1TenantResponse struct {
	Tenant  *tenant `json:"tenant"`
	Version string  `json:"version"`
}

type v1TenantSliceResponse struct {
	Tenants tenantSlice `json:"tenants"`
	Version string      `json:"version"`
	PaginationParams
}

func v1TenantCreatedResponse(c echo.Context, t *models.Tenant) error {
	return c.JSON(http.StatusCreated, v1TenantResponse{
		Tenant:  v1Tenant(t),
		Version: apiVersion,
	})
}

func v1TenantsResponse(c echo.Context, ts []*models.Tenant, pagination PaginationParams) error {
	return c.JSON(http.StatusOK, v1TenantSliceResponse{
		Tenants:          v1TenantSlice(ts),
		Version:          apiVersion,
		PaginationParams: pagination,
	})
}

func v1TenantGetResponse(c echo.Context, t *models.Tenant) error {
	return c.JSON(http.StatusOK, v1TenantResponse{
		Tenant:  v1Tenant(t),
		Version: apiVersion,
	})
}

func v1TenantNotFoundResponse(c echo.Context, err error) error {
	return c.JSON(http.StatusNotFound, struct {
		Version string `json:"version"`
		Message string `json:"message"`
		Error   string `json:"error"`
		Status  int    `json:"status"`
	}{
		Version: apiVersion,
		Message: "tenant not found",
		Error:   err.Error(),
		Status:  http.StatusNotFound,
	})
}

func v1BadRequestResponse(c echo.Context, err error) error {
	return c.JSON(http.StatusBadRequest, struct {
		Version string `json:"version"`
		Message string `json:"message"`
		Error   string `json:"error"`
		Status  int    `json:"status"`
	}{
		Version: apiVersion,
		Message: "bad request",
		Error:   err.Error(),
		Status:  http.StatusBadRequest,
	})
}

func v1InternalServerErrorResponse(c echo.Context, err error) error {
	return c.JSON(http.StatusInternalServerError, struct {
		Version string `json:"version"`
		Message string `json:"message"`
		Error   string `json:"error"`
		Status  int    `json:"status"`
	}{
		Version: apiVersion,
		Message: "internal server error",
		Error:   err.Error(),
		Status:  http.StatusInternalServerError,
	})
}
