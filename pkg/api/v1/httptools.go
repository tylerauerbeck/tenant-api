package api

import (
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// parseUUID parses and validates a UUID from the request path if the path param is found
func parseUUID(c echo.Context, path string) (string, error) {
	var id string
	if err := echo.PathParamsBinder(c).String(path, &id).BindError(); err != nil {
		return "", err
	}

	if id != "" {
		if _, err := uuid.Parse(id); err != nil {
			return "", ErrInvalidUUID
		}

		return id, nil
	}

	return "", ErrUUIDNotFound
}
