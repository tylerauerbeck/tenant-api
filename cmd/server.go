/*
Copyright © 2022 The Infratographer Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.infratographer.com/x/crdbx"
	"go.infratographer.com/x/otelx"
	"go.infratographer.com/x/versionx"

	"go.infratographer.com/identityapi/internal/config"
	"go.infratographer.com/identityapi/pkg/api"
	"go.infratographer.com/identityapi/pkg/echox"
)

var (
	APIDefaultListen = "0.0.0.0:7601"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "starts the permission api server",
	Run: func(cmd *cobra.Command, args []string) {
		serve(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(serverCmd)

	// ginx.MustViperFlags(viper.GetViper(), serverCmd.Flags(), APIDefaultListen)
	echox.MustViperFlags(viper.GetViper(), serverCmd.Flags(), APIDefaultListen)
	otelx.MustViperFlags(viper.GetViper(), serverCmd.Flags())
	crdbx.MustViperFlags(viper.GetViper(), serverCmd.Flags())
}

func serve(ctx context.Context) {
	err := otelx.InitTracer(config.AppConfig.Tracing, appName, logger)
	if err != nil {
		logger.Fatalw("unable to initialize tracing system", "error", err)
	}

	db, err := crdbx.NewDB(config.AppConfig.CRDB, config.AppConfig.Tracing.Enabled)
	if err != nil {
		logger.Fatalw("unable to initialize crdb client", "error", err)
	}

	e := echox.NewServer(logger.Desugar(), config.AppConfig.Server, versionx.BuildDetails())
	r := api.NewRouter(db, logger)

	r.Routes(e)

	// s = s.AddHandler(r).
	// 	AddReadinessCheck("crdb", db.PingContext)

	e.Logger.Fatal(e.Start(config.AppConfig.Server.Listen))
}
