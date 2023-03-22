package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	"github.com/pressly/goose/v3"
	dbm "go.infratographer.com/tenant-api/db"
	"go.infratographer.com/tenant-api/pkg/echox"
	"go.infratographer.com/tenant-api/pkg/jwtauth"
	"go.infratographer.com/x/crdbx"
	"go.uber.org/zap"
)

type testServer struct {
	*httptest.Server
	client   *http.Client
	closeFns []func()
}

func (t *testServer) close() {
	if t == nil {
		return
	}

	for _, fn := range t.closeFns {
		fn()
	}
}

func (t *testServer) Request(method, path string, headers http.Header, body io.Reader, out interface{}) (*http.Response, error) {
	return t.RequestWithClient(t.client, method, path, headers, body, out)
}

func (t *testServer) RequestWithClient(client *http.Client, method, path string, headers http.Header, body io.Reader, out interface{}) (*http.Response, error) {
	uri, err := buildURL(t.Server.URL, path)
	if err != nil {
		return nil, err
	}

	return httpRequest(client, method, uri, headers, body, out)
}

func buildURL(baseURL, path string) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	up, err := url.Parse(path)
	if err != nil {
		return "", err
	}

	u.Path += up.Path

	query := u.Query()

	for k, v := range up.Query() {
		query[k] = v
	}

	u.RawQuery = query.Encode()

	return u.String(), nil
}

func httpRequest(client *http.Client, method, uri string, headers http.Header, body io.Reader, out interface{}) (*http.Response, error) {
	req, err := http.NewRequestWithContext(context.Background(), method, uri, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header = headers

	resp, err := client.Do(req)
	if err != nil {
		return resp, err
	}

	if out != nil {
		err = json.NewDecoder(resp.Body).Decode(&out)
		resp.Body.Close()
	}

	return resp, err
}

type testServerConfig struct {
	client *http.Client
	auth   *jwtauth.AuthConfig
}

func newTestServer(config *testServerConfig) (*testServer, error) {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)

	logger, err := loggerConfig.Build()
	if err != nil {
		return nil, err
	}

	ts := new(testServer)

	if config == nil {
		config = new(testServerConfig)
	}

	if config.client != nil {
		ts.client = config.client
	} else {
		ts.client = http.DefaultClient
	}

	srv, err := testserver.NewTestServer()
	if err != nil {
		return nil, err
	}

	ts.closeFns = append(ts.closeFns, srv.Stop)

	if err := srv.WaitForInit(); err != nil {
		ts.Close()

		return nil, err
	}

	dbURL := srv.PGURL()

	// Reset Path so we can use the database in general
	dbURL.Path = "/"

	db, err := crdbx.NewDB(crdbx.Config{URI: dbURL.String()}, false)
	if err != nil {
		ts.Close()

		return nil, err
	}

	goose.SetBaseFS(dbm.Migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		ts.Close()

		return nil, err
	}

	if err := goose.Up(db, "migrations"); err != nil {
		ts.Close()

		return nil, err
	}

	e := echox.NewServer()

	var auth *jwtauth.Auth

	if config.auth != nil {
		auth, err = jwtauth.NewAuth(*config.auth)
		if err != nil {
			ts.Close()

			return nil, err
		}
	}

	router := NewRouter(db, logger, auth)

	router.Routes(e)

	ts.Server = httptest.NewServer(e)

	ts.closeFns = append(ts.closeFns, ts.Server.Close)

	return ts, nil
}
