package jwtauth

import (
	"github.com/golang-jwt/jwt/v4"
	"github.com/labstack/echo/v4"
)

// successHandler retrieves the user and sets the ActorKey to the jwt subject.
func (a *Auth) successHandler(c echo.Context) {
	if a.jwtConfig.SuccessHandler != nil {
		defer a.jwtConfig.SuccessHandler(c)
	}

	token, ok := c.Get("user").(*jwt.Token)
	if !ok {
		a.logger.Warn("jwt user is not jwt.Token")

		return
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		a.logger.Warn("jwt user claims are not jwt.MapClaims type")

		return
	}

	if subject, ok := claims["sub"]; ok {
		c.Set(ActorKey, subject)
	}
}

// Actor retrieves the ActorKey from echo Context.
func Actor(c echo.Context) string {
	if actor, ok := c.Get(ActorKey).(string); ok {
		return actor
	}

	return ""
}
