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
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.infratographer.com/x/goosex"
	"go.infratographer.com/x/loggingx"
	"go.infratographer.com/x/versionx"
	"go.uber.org/zap"

	"go.infratographer.com/identity-api/internal/config"
	"go.infratographer.com/identity-api/internal/dbschema"
)

var (
	appName = "identityapi"
	cfgFile string
	logger  *zap.SugaredLogger
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "infra-identity-api",
	Short: "Infratographer Identity API Service",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is /etc/infratographer/identity-api.yaml)")
	loggingx.MustViperFlags(rootCmd.PersistentFlags())

	// Add migrate command
	goosex.RegisterCobraCommand(rootCmd, func() {
		goosex.SetBaseFS(dbschema.Migrations)
		goosex.SetLogger(logger)
		goosex.SetDBURI(config.AppConfig.CRDB.GetURI())
	})
	// Add version command
	versionx.RegisterCobraCommand(rootCmd, func() { versionx.PrintVersion(logger) })
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("/etc/infratographer/")
		viper.SetConfigType("yaml")
		viper.SetConfigName("identity-api")
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetEnvPrefix(appName)
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	err := viper.ReadInConfig()

	logger = loggingx.InitLogger(appName, config.AppConfig.Logging)

	if err == nil {
		logger.Infow("using config file",
			"file", viper.ConfigFileUsed(),
		)
	}

	err = viper.Unmarshal(&config.AppConfig)
	if err != nil {
		logger.Fatalw("unable to decode app config", "error", err)
	}
}
