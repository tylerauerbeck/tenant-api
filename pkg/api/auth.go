package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/golang-jwt/jwt/v4/request"
	"github.com/labstack/echo/v4"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"go.infratographer.com/identity-api/internal/models"
)

var (
	errAuthCheckInvalidIssuer = errors.New("invalid issuer")
	errAuthCheckUnknownKey    = errors.New("unable to find key")
	errAuthCheckJWKSMissing   = errors.New("issuer missing jwks_uri from discovery endpoint")
)

func (r *Router) authRequest(c echo.Context) error {
	claims := jwt.MapClaims{}
	userInfo := userClaims{}

	_, err := request.ParseFromRequest(c.Request(), request.BearerExtractor{}, func(t *jwt.Token) (interface{}, error) {
		var issuer *models.OidcIssuer

		issClaim := claims["iss"].(string)
		audClaim := []string{}

		switch v := claims["aud"].(type) {
		case string:
			audClaim = append(audClaim, v)
		case []string:
			audClaim = append(audClaim, v...)
		}

		for _, aud := range audClaim {
			i, err := r.findOIDCIssuer(c.Request().Context(), issClaim, aud)
			if err != nil && errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}

			if i != nil {
				issuer = i
				break
			}
		}

		if issuer == nil {
			return nil, errAuthCheckInvalidIssuer
		}

		userInfo = newUserClaims(claims, issuer)

		keyID := t.Header["kid"]

		return lookupKey(issClaim, keyID.(string))
	}, request.WithClaims(claims))

	if err != nil {
		return err
	}

	u, err := r.getUserFromClaims(context.TODO(), userInfo)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, v1User(u))
}

var keyMap map[string]jwk.Set

func init() {
	keyMap = map[string]jwk.Set{}
}

type userClaims struct {
	issuer   *models.OidcIssuer
	audience string
	subject  string
	name     string
	email    string
}

func newUserClaims(c map[string]interface{}, issuer *models.OidcIssuer) userClaims {
	claims := userClaims{
		issuer: issuer,
	}

	if c[issuer.SubjectClaim] != nil {
		claims.subject = c[issuer.SubjectClaim].(string)
	}

	if c[issuer.NameClaim] != nil {
		claims.name = c[issuer.NameClaim].(string)
	}

	if c[issuer.EmailClaim] != nil {
		claims.email = c[issuer.EmailClaim].(string)
	}

	return claims
}

func (r *Router) getUserFromClaims(ctx context.Context, claims userClaims) (*models.User, error) {
	qms := []qm.QueryMod{

		models.UserWhere.OidcSubject.EQ(claims.subject),
	}

	u, err := models.Users(qms...).One(ctx, r.db)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}

		u = &models.User{
			OidcSubject:  claims.subject,
			OidcIssuerID: claims.issuer.ID,
			Name:         null.NewString(claims.name, claims.name != ""),
			Email:        null.NewString(claims.email, claims.email != ""),
		}

		boil.DebugMode = true

		if err := u.Insert(ctx, r.db, boil.Infer()); err != nil {
			return nil, err
		}

		boil.DebugMode = false

		return u, nil
	}

	return u, nil
}

func (r *Router) findOIDCIssuer(ctx context.Context, uri string, aud string) (*models.OidcIssuer, error) {
	where := models.OidcIssuerWhere

	return models.OidcIssuers(
		where.URI.EQ(uri),
		where.Audience.EQ(aud),
	).One(ctx, r.db)
}

func lookupKey(issuer string, kid string) (k interface{}, err error) {
	ctx := context.Background()

	jwkset := keyMap[issuer]
	if jwkset == nil {
		s, err := buildJWKSet(ctx, issuer)
		if err != nil {
			return nil, err
		}

		keyMap[issuer] = s
		jwkset = s
	}

	key, ok := jwkset.LookupKeyID(kid)
	if ok {
		return k, key.Raw(&k)
	}

	// We didn't find the key, the spec says we should update from the jwks url before failing

	jwkset, err = buildJWKSet(ctx, issuer)
	if err != nil {
		return nil, err
	}

	keyMap[issuer] = jwkset

	key, ok = jwkset.LookupKeyID(kid)
	if ok {
		return k, key.Raw(&k)
	}

	return nil, errAuthCheckUnknownKey
}

func buildJWKSet(ctx context.Context, issuer string) (jwk.Set, error) {
	uri, err := fetchJWKSURL(ctx, issuer)
	if err != nil {
		return nil, err
	}

	return jwk.Fetch(ctx, uri)
}

func fetchJWKSURL(ctx context.Context, issuer string) (string, error) {
	uri, err := url.JoinPath(issuer, ".well-known", "openid-configuration")
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	var m map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&m); err != nil {
		return "", err
	}

	jwksURL, ok := m["jwks_uri"]
	if !ok {
		return "", errAuthCheckJWKSMissing
	}

	return jwksURL.(string), nil
}

func v1User(u *models.User) any {
	return struct {
		ID        string     `json:"id"`
		URN       string     `json:"urn"`
		Name      *string    `json:"name"`
		Email     *string    `json:"email,omitempty"`
		IssuerID  string     `json:"issuer_id"`
		CreatedAt time.Time  `json:"created_at"`
		UpdatedAt time.Time  `json:"updated_at"`
		DeletedAt *time.Time `json:"deleted_at,omitempty"`
	}{
		ID:        u.ID,
		URN:       "urn:infratographer:user:" + u.ID,
		Name:      u.Name.Ptr(),
		Email:     u.Email.Ptr(),
		IssuerID:  u.OidcIssuerID,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
		DeletedAt: u.DeletedAt.Ptr(),
	}
}
