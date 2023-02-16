package api

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"go.infratographer.com/tenant-api/internal/models"
)

// ErrorResponse represents the data that the server will return on any given call
type ErrorResponse struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

var (
	ErrResourceNotFound      = errors.New("resource not found")
	ErrSearchNotFound        = errors.New("search term not found")
	ErrResourceAlreadyExists = errors.New("resource already exists")
)

func createdResponse(c *gin.Context, resource any) {
	c.Header("Location", uriFor(resource))
	c.JSON(http.StatusCreated, gin.H{
		"message": typeNameFor(resource) + " created",
		"id":      slugFor(resource),
	})
}

func typeNameFor(obj any) string {
	switch obj.(type) {
	case *models.Tenant:
		return "organization unit"
	default:
		return ""
	}
}

func uriFor(obj any) string {
	switch o := obj.(type) {
	case *models.Tenant:
		return fmt.Sprintf("/organization-units/%s", slugFor(o))
	default:
		return ""
	}
}

func slugFor(obj any) string {
	switch o := obj.(type) {
	case *models.Tenant:
		return o.ID
	default:
		return ""
	}
}

// func addPaginationHeaders(c *gin.Context, pag *Pagination) {

// }
