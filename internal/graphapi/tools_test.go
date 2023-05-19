package graphapi_test

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"entgo.io/ent/dialect"
	"github.com/99designs/gqlgen/graphql/handler"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	natssrv "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go"
	"go.infratographer.com/x/echojwtx"
	"go.infratographer.com/x/goosex"
	"go.infratographer.com/x/testing/containersx"
	"go.uber.org/zap"

	"go.infratographer.com/tenant-api/db"
	ent "go.infratographer.com/tenant-api/internal/ent/generated"
	"go.infratographer.com/tenant-api/internal/ent/generated/pubsubhooks"
	"go.infratographer.com/tenant-api/internal/graphapi"
	"go.infratographer.com/tenant-api/internal/pubsub"
	"go.infratographer.com/tenant-api/internal/testclient"
)

var (
	TestDBURI      = os.Getenv("TENANTAPI_TESTDB_URI")
	EntClient      *ent.Client
	DBContainer    *containersx.DBContainer
	DBURI          string
	NatsTestClient *pubsub.Client
)

func TestMain(m *testing.M) {
	// setup the database if needed
	setupDB()
	// run the tests
	code := m.Run()
	// teardown the database
	teardownDB()
	// return the test response code
	os.Exit(code)
}

func parseDBURI(ctx context.Context) (string, string, *containersx.DBContainer) {
	switch {
	// if you don't pass in a database we default to an in memory sqlite
	case TestDBURI == "":
		return dialect.SQLite, "file:ent?mode=memory&cache=shared&_fk=1", nil
	case strings.HasPrefix(TestDBURI, "sqlite://"):
		return dialect.SQLite, strings.TrimPrefix(TestDBURI, "sqlite://"), nil
	case strings.HasPrefix(TestDBURI, "postgres://"), strings.HasPrefix(TestDBURI, "postgresql://"):
		return dialect.Postgres, TestDBURI, nil
	case strings.HasPrefix(TestDBURI, "docker://"):
		dbImage := strings.TrimPrefix(TestDBURI, "docker://")

		switch {
		case strings.HasPrefix(dbImage, "cockroach"), strings.HasPrefix(dbImage, "cockroachdb"), strings.HasPrefix(dbImage, "crdb"):
			cntr, err := containersx.NewCockroachDB(ctx, dbImage)
			errPanic("error starting db test container", err)

			return dialect.Postgres, cntr.URI, cntr
		case strings.HasPrefix(dbImage, "postgres"):
			cntr, err := containersx.NewPostgresDB(ctx, dbImage)
			errPanic("error starting db test container", err)

			return dialect.Postgres, cntr.URI, cntr
		default:
			panic("invalid testcontainer URI, uri: " + TestDBURI)
		}

	default:
		panic("invalid DB URI, uri: " + TestDBURI)
	}
}

func setupDB() {
	// don't setup the datastore if we already have one
	if EntClient != nil {
		return
	}

	ctx := context.Background()

	ns, err := pubsub.StartNatsServer()
	if err != nil {
		errPanic("failed to start nats server", err)
	}

	natsClient, err := newNatsClient(ns)
	if err != nil {
		errPanic("failed to generate nats client", err)
	}

	NatsTestClient = natsClient

	dia, uri, cntr := parseDBURI(ctx)

	c, err := ent.Open(dia, uri, ent.Debug(), ent.PubsubClient(natsClient))
	if err != nil {
		errPanic("failed terminating test db container after failing to connect to the db", cntr.Container.Terminate(ctx))
		errPanic("failed opening connection to database:", err)
	}

	switch dia {
	case dialect.SQLite:
		// Run automatic migrations for SQLite
		errPanic("failed creating db scema", c.Schema.Create(ctx))
	case dialect.Postgres:
		log.Println("Running database migrations")
		goosex.MigrateUp(uri, db.Migrations)
	}

	EntClient = c
}

func entClientWithPubsubHooks() *ent.Client {
	newClient := EntClient
	pubsubhooks.PubsubHooks(newClient)

	return newClient
}

func teardownDB() {
	ctx := context.Background()

	if EntClient != nil {
		errPanic("teardown failed to close database connection", EntClient.Close())
	}

	if DBContainer != nil {
		errPanic("teardown failed to terminate test db container", DBContainer.Container.Terminate(ctx))
	}
}

func errPanic(msg string, err error) {
	if err != nil {
		log.Panicf("%s err: %s", msg, err.Error())
	}
}

func graphTestClient(entClient *ent.Client) testclient.TestClient {
	return testclient.NewClient(&http.Client{Transport: localRoundTripper{handler: handler.NewDefaultServer(
		graphapi.NewExecutableSchema(
			graphapi.Config{Resolvers: graphapi.NewResolver(entClient, zap.NewNop().Sugar())},
		))}}, "graph")
}

// localRoundTripper is an http.RoundTripper that executes HTTP transactions
// by using handler directly, instead of going over an HTTP connection.
type localRoundTripper struct {
	handler http.Handler
}

func (l localRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	w := httptest.NewRecorder()
	// set the actor to "testing-roundtrip-actor"
	req := r.WithContext(context.WithValue(r.Context(), echojwtx.ActorCtxKey, "testing-roundtrip-actor"))
	l.handler.ServeHTTP(w, req)

	return w.Result(), nil
}

func newNatsClient(srv *natssrv.Server) (*pubsub.Client, error) {
	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		// errPanic("teardown failed to terminate test db container", DBContainer.Container.Terminate(ctx))
		return &pubsub.Client{}, err
	}

	js, err := nc.JetStream()
	if err != nil {
		return &pubsub.Client{}, err
	}

	_, err = js.AddStream(&nats.StreamConfig{
		Name:     "tenant-api",
		Subjects: []string{"com.infratographer.events.>", "com.infratographer.changes.>"},
	})
	if err != nil {
		return &pubsub.Client{}, err
	}

	client := pubsub.NewClient(pubsub.WithJetreamContext(js),
		pubsub.WithLogger(zap.NewNop().Sugar()),
		pubsub.WithStreamName("tenant-api"),
		pubsub.WithSubjectPrefix("com.infratographer"),
		pubsub.WithSource("tenant-api-test"),
	)

	return client, nil
}
