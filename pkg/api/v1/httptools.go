package api

import (
	"github.com/labstack/echo/v4"
	"go.infratographer.com/x/gidx"
)

// parseID parses and validates a GID from the request path if the path param is found
func parseID(c echo.Context, path string) (gidx.PrefixedID, error) {
	var id string
	if err := echo.PathParamsBinder(c).String(path, &id).BindError(); err != nil {
		return "", err
	}

	if id != "" {
		gid, err := gidx.Parse(id)

		if err != nil {
			return "", ErrInvalidID
		}

		return gid, nil
	}

	return "", ErrIDNotFound
}
