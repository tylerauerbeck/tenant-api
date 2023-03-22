package jwtauth

import (
	"github.com/MicahParks/keyfunc"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

const (
	ActorKey = "actor"
)

func noopMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return next
}

// AuthConfig provides configuration for JWT validation using JWKS.
type AuthConfig struct {
	// Logger defines the auth logger to use.
	Logger *zap.Logger

	// JWKSURI is the full URI to a JWKS json path.
	JWKSURI string

	// JWTConfig configuration for handling JWT validation.
	JWTConfig echojwt.Config

	// KeyFuncOptions configuration for fetching JWKS.
	KeyFuncOptions keyfunc.Options
}

// Auth handles JWT Authentication as echo middleware.
type Auth struct {
	logger *zap.Logger

	jwtConfig echojwt.Config

	middleware echo.MiddlewareFunc
}

func (a *Auth) setup(config AuthConfig) error {
	if config.Logger != nil {
		a.logger = config.Logger
	} else {
		a.logger = zap.NewNop()
	}

	jwks, err := keyfunc.Get(config.JWKSURI, config.KeyFuncOptions)
	if err != nil {
		return err
	}

	a.jwtConfig = config.JWTConfig

	jwtConfig := &config.JWTConfig
	jwtConfig.KeyFunc = jwks.Keyfunc
	jwtConfig.SuccessHandler = a.successHandler

	middleware, err := jwtConfig.ToMiddleware()
	if err != nil {
		return err
	}

	a.middleware = middleware

	return nil
}

// Middleware returns echo middleware for validation jwt tokens.
func (a *Auth) Middleware() echo.MiddlewareFunc {
	if a == nil || a.middleware == nil {
		return noopMiddleware
	}

	return a.middleware
}

// NewAuth creates a new auth middleware handler for JWTs using JWKS.
func NewAuth(config AuthConfig) (*Auth, error) {
	auth := new(Auth)

	if err := auth.setup(config); err != nil {
		return nil, err
	}

	return auth, nil
}
