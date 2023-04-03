package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/cockroachdb/cockroach-go/v2/testserver"
	natssrv "github.com/nats-io/nats-server/v2/server"
	nats "github.com/nats-io/nats.go"
	"github.com/pressly/goose/v3"
	dbm "go.infratographer.com/tenant-api/db"
	"go.infratographer.com/tenant-api/internal/pubsub"
	"go.infratographer.com/tenant-api/pkg/echox"
	"go.infratographer.com/tenant-api/pkg/jwtauth"
	"go.infratographer.com/x/crdbx"
	"go.uber.org/zap"
)

type testServer struct {
	*httptest.Server
	logger   *zap.Logger
	client   *http.Client
	nats     *natssrv.Server
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

func newTestServer(t *testing.T, config *testServerConfig) (*testServer, error) {
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)

	logger, err := loggerConfig.Build()
	if err != nil {
		return nil, err
	}

	ts := &testServer{
		logger: logger,
	}

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

	ts.nats = newNatsTestServer(t, "tenant-api-test", "com.infratographer.events.>")

	ts.closeFns = append(ts.closeFns, ts.nats.Shutdown)

	e := echox.NewServer()

	if config.auth != nil {
		auth, err := jwtauth.NewAuth(*config.auth)
		if err != nil {
			ts.Close()

			return nil, err
		}

		e.Use(auth.Middleware())
	}

	router := NewRouter(db, logger, newPubSubClient(t, logger, ts.nats.ClientURL()))

	router.Routes(e)

	ts.Server = httptest.NewServer(e)

	ts.closeFns = append(ts.closeFns, ts.Server.Close)

	return ts, nil
}

// newNatsTestServer creates a new nats server for testing and generates a new
// stream. The returned server should be Shutdown() when testing is done.
func newNatsTestServer(t *testing.T, stream string, subs ...string) *natssrv.Server {
	srv, err := pubsub.StartNatsServer()
	if err != nil {
		t.Error(err)
	}

	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Error(err)
	}

	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		t.Error(err)
	}

	if _, err = js.AddStream(&nats.StreamConfig{
		Name:     stream,
		Subjects: subs,
		Storage:  nats.MemoryStorage,
	}); err != nil {
		t.Error(err)
	}

	return srv
}

func newPubSubClient(t *testing.T, logger *zap.Logger, url string) *pubsub.Client {
	nc, err := nats.Connect(url)
	if err != nil {
		// fail open on nats
		t.Error(err)
	}

	js, err := nc.JetStream()
	if err != nil {
		// fail open on nats
		t.Error(err)
	}

	return pubsub.NewClient(
		pubsub.WithJetreamContext(js),
		pubsub.WithLogger(logger),
		pubsub.WithStreamName("tenant-api-test"),
		pubsub.WithSubjectPrefix("com.infratographer.events"),
	)
}
