package cmd

import (
	"context"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nats-io/nats.go"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.infratographer.com/x/crdbx"
	"go.infratographer.com/x/echojwtx"
	"go.infratographer.com/x/echox"
	"go.infratographer.com/x/otelx"
	"go.infratographer.com/x/versionx"
	"go.uber.org/zap"

	"go.infratographer.com/tenant-api/internal/config"
	ent "go.infratographer.com/tenant-api/internal/ent/generated"
	"go.infratographer.com/tenant-api/internal/graphapi"
	"go.infratographer.com/tenant-api/internal/pubsub"
)

// APIDefaultListen defines the default listening address for the tenant-api.
const APIDefaultListen = ":7902"

var (
	enablePlayground bool
	serveDevMode     bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start Tenant API",
	Run: func(cmd *cobra.Command, args []string) {
		serve(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	echox.MustViperFlags(viper.GetViper(), serveCmd.Flags(), APIDefaultListen)
	echojwtx.MustViperFlags(viper.GetViper(), serveCmd.Flags())

	// only available as a CLI arg because it shouldn't be something that could accidentially end up in a config file or env var
	serveCmd.Flags().BoolVar(&serveDevMode, "dev", false, "dev mode: enables playground, disables all auth checks, sets CORS to allow all, pretty logging, etc.")
	serveCmd.Flags().BoolVar(&enablePlayground, "playground", false, "enable the graph playground")
}

func serve(ctx context.Context) {
	if serveDevMode {
		enablePlayground = true
		config.AppConfig.Logging.Debug = true
		config.AppConfig.Logging.Pretty = true
		config.AppConfig.Server.WithMiddleware(middleware.CORS())
		// this is a hack, echojwt needs to be updated to go into AppConfig
		viper.Set("oidc.enabled", false)
	}

	js, natsClose, err := newJetstreamConnection()
	if err != nil {
		logger.Fatal("failed to create NATS jetstream connection", zap.Error(err))
	}

	defer natsClose()

	pubsubClient := pubsub.NewClient(pubsub.WithJetreamContext(js),
		pubsub.WithLogger(logger),
		pubsub.WithStreamName(viper.GetString("nats.stream-name")),
		pubsub.WithSubjectPrefix("com.infratographer"),
		pubsub.WithSource("tenant-api"),
	)

	err = otelx.InitTracer(config.AppConfig.Tracing, appName, logger)
	if err != nil {
		logger.Fatal("unable to initialize tracing system", zap.Error(err))
	}

	db, err := crdbx.NewDB(config.AppConfig.CRDB, config.AppConfig.Tracing.Enabled)
	if err != nil {
		logger.Fatal("unable to initialize crdb client", zap.Error(err))
	}

	defer db.Close()

	entDB := entsql.OpenDB(dialect.Postgres, db)

	cOpts := []ent.Option{ent.Driver(entDB), ent.PubsubClient(pubsubClient)}

	if config.AppConfig.Logging.Debug {
		cOpts = append(cOpts,
			ent.Log(logger.Named("ent").Debugln),
			ent.Debug(),
		)
	}

	client := ent.NewClient(cOpts...)

	srv, err := echox.NewServer(logger.Desugar(), echox.ConfigFromViper(viper.GetViper()), versionx.BuildDetails())
	if err != nil {
		logger.Fatal("failed to initialize new server", zap.Error(err))
	}

	var middleware []echo.MiddlewareFunc

	if config, err := echojwtx.AuthConfigFromViper(viper.GetViper()); err != nil {
		logger.Fatal("failed to initialize jwt authentication", zap.Error(err))
	} else if config != nil {
		config.JWTConfig.Skipper = echox.SkipDefaultEndpoints

		auth, err := echojwtx.NewAuth(ctx, *config)
		if err != nil {
			logger.Fatal("failed to initialize jwt authentication", zap.Error(err))
		}

		middleware = append(middleware, auth.Middleware())
	}

	r := graphapi.NewResolver(client, logger.Named("resolvers"))
	handler := r.Handler(enablePlayground, middleware)

	srv.AddHandler(handler)

	// TODO: we should have a database check
	// srv.AddReadinessCheck("database", r.DatabaseCheck)

	if err := srv.Run(); err != nil {
		logger.Fatal("failed to run server", zap.Error(err))
	}
}

func newJetstreamConnection() (nats.JetStreamContext, func(), error) {
	opts := []nats.Option{nats.Name(appName)}

	if viper.GetBool("debug") {
		logger.Debug("enabling development settings")

		opts = append(opts, nats.Token(viper.GetString("nats.token")))
	} else {
		opts = append(opts, nats.UserCredentials(viper.GetString("nats.creds-file")))
	}

	nc, err := nats.Connect(viper.GetString("nats.url"), opts...)
	if err != nil {
		return nil, nil, err
	}

	js, err := nc.JetStream()
	if err != nil {
		return nil, nil, err
	}

	return js, nc.Close, nil
}
