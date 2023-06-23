package cmd

import (
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/spf13/cobra"
	"go.infratographer.com/x/crdbx"
	"go.infratographer.com/x/events"
	"go.infratographer.com/x/otelx"
	"go.uber.org/zap"

	"go.infratographer.com/tenant-api/internal/config"
	ent "go.infratographer.com/tenant-api/internal/ent/generated"
	"go.infratographer.com/tenant-api/internal/ent/generated/eventhooks"
)

var tenantCmd = &cobra.Command{
	Use:   "tenant",
	Short: "Tenant management",
}

func initializeGraphClient() (*ent.Client, func()) {
	publisher, err := events.NewPublisher(config.AppConfig.Events.Publisher)
	if err != nil {
		logger.Fatal("unable to initialize event publisher", zap.Error(err))
	}

	err = otelx.InitTracer(config.AppConfig.Tracing, appName, logger)
	if err != nil {
		logger.Fatal("unable to initialize tracing system", zap.Error(err))
	}

	db, err := crdbx.NewDB(config.AppConfig.CRDB, config.AppConfig.Tracing.Enabled)
	if err != nil {
		logger.Fatal("unable to initialize crdb client", zap.Error(err))
	}

	entDB := entsql.OpenDB(dialect.Postgres, db)

	cOpts := []ent.Option{ent.Driver(entDB), ent.EventsPublisher(publisher)}

	if config.AppConfig.Logging.Debug {
		cOpts = append(cOpts,
			ent.Log(logger.Named("ent").Debugln),
			ent.Debug(),
		)
	}

	client := ent.NewClient(cOpts...)

	eventhooks.EventHooks(client)

	return client, func() { db.Close(); client.Close() }
}

func init() {
	rootCmd.AddCommand(tenantCmd)
}
