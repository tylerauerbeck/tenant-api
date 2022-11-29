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
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/volatiletech/null/v8"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"go.infratographer.com/permissionapi/pkg/pubsubx"
	"go.infratographer.com/x/crdbx"
	"go.infratographer.com/x/viperx"

	"go.infratographer.com/identityapi/internal/config"
	"go.infratographer.com/identityapi/internal/models"
)

var createCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "bootstrap an initial user, tenent, and oidc provider",
	Run: func(cmd *cobra.Command, args []string) {
		bootstrap(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(createCmd)

	v := viper.GetViper()
	cmdFlags := createCmd.Flags()

	crdbx.MustViperFlags(v, cmdFlags)

	cmdFlags.String("tenant-name", "", "Tenant Name")
	viperx.MustBindFlag(v, "bootstrap.tenant.name", cmdFlags.Lookup("tenant-name"))

	cmdFlags.String("issuer-uri", "", "OIDC Issuer URI")
	viperx.MustBindFlag(v, "bootstrap.issuer.uri", cmdFlags.Lookup("issuer-uri"))
	cmdFlags.String("issuer-aud", "", "OIDC Issuer Audience")
	viperx.MustBindFlag(v, "bootstrap.issuer.audience", cmdFlags.Lookup("issuer-aud"))
	cmdFlags.String("issuer-claims-subject", "sub", "JWT claim to use for the user subject")
	viperx.MustBindFlag(v, "bootstrap.issuer.claims.subject", cmdFlags.Lookup("issuer-claims-subject"))
	cmdFlags.String("issuer-claims-email", "email", "JWT claim to use for the user email")
	viperx.MustBindFlag(v, "bootstrap.issuer.claims.email", cmdFlags.Lookup("issuer-claims-email"))
	cmdFlags.String("issuer-claims-name", "name", "JWT claim to use for the user name")
	viperx.MustBindFlag(v, "bootstrap.issuer.claims.name", cmdFlags.Lookup("issuer-claims-name"))

	cmdFlags.String("user-subject", "name", "JWT subject for the bootstrap user, MUST match whats in the issuier-claims-subject for this user")
	viperx.MustBindFlag(v, "bootstrap.user.subject", cmdFlags.Lookup("user-subject"))
	cmdFlags.String("user-email", "email", "user email address")
	viperx.MustBindFlag(v, "bootstrap.user.email", cmdFlags.Lookup("user-email"))
}

func bootstrap(ctx context.Context) {
	cfg := config.AppConfig.Bootstrap

	fields := [][2]string{
		{cfg.Tenant.Name, "Tenant Name"},
		{cfg.Issuer.URI, "OIDC Issuer URI"},
		{cfg.Issuer.Audience, "OIDC Issuer Audience"},
		{cfg.Issuer.Claims.Subject, "OIDC Issuer Claims Subject"},
		{cfg.Issuer.Claims.Email, "OIDC Issuer Claims Email"},
		{cfg.Issuer.Claims.Name, "OIDC Issuer Claims Name"},
		{cfg.User.Subject, "User JWT Subject"},
		{cfg.User.Email, "User Email"},
	}

	invalid := false

	for _, f := range fields {
		if f[0] == "" {
			invalid = true

			fmt.Printf("missing: %s\n", f[1])
		}
	}

	if invalid {
		fmt.Println("Please run bootstrap again with the missing fields set.")
		os.Exit(1)
	}

	fmt.Printf("Bootstrap Info:\n\n")
	fmt.Printf("Tenant:\n\tName: %s\n", cfg.Tenant.Name)
	fmt.Printf("Issuer:\n\tURI: %s\n\tAudience: %s\n", cfg.Issuer.URI, cfg.Issuer.Audience)
	fmt.Printf("\tJWT Claim Values:\n\t\tSubject: %s\n\t\tEmail: %s\n\t\tName: %s\n", cfg.Issuer.Claims.Subject, cfg.Issuer.Claims.Email, cfg.Issuer.Claims.Name)
	fmt.Printf("User:\n\tSubject: %s\n\tEmail: %s\n", cfg.User.Subject, cfg.User.Email)
	fmt.Printf("\n\n")

	if !askForConfirmation("Confirm that this is the correct bootstrap data") {
		fmt.Println("exiting...")
		os.Exit(1)
	}

	db, err := crdbx.NewDB(config.AppConfig.CRDB, config.AppConfig.Tracing.Enabled)
	if err != nil {
		logger.Fatalw("unable to initialize crdb client", "error", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		logger.Fatalw("failed to begin database transaction", "error", err)
	}
	defer tx.Rollback()

	t := &models.Tenant{
		Name: cfg.Tenant.Name,
	}
	iss := &models.OidcIssuer{
		URI:          cfg.Issuer.URI,
		Audience:     cfg.Issuer.Audience,
		SubjectClaim: cfg.Issuer.Claims.Subject,
		EmailClaim:   cfg.Issuer.Claims.Email,
		Name:         cfg.Issuer.Claims.Name,
	}
	u := &models.User{
		OidcSubject: cfg.User.Subject,
		Email:       null.StringFrom(cfg.User.Email),
	}

	if err := t.Insert(ctx, tx, boil.Infer()); err != nil {
		logger.Fatalw("failed to create tenant", "error", err)
	}

	if err := t.AddOidcIssuers(ctx, tx, true, iss); err != nil {
		logger.Fatalw("failed to create issuer", "error", err)
	}

	if err := iss.AddUsers(ctx, tx, true, u); err != nil {
		logger.Fatalw("failed to create user", "error", err)
	}

	if err := pubsubx.HackySendMsg(ctx, "tenant.added", &pubsubx.Message{
		SubjectURN: "urn:infratographer:tenant:" + t.ID,
		EventType:  "tenant.added",
		ActorURN:   "urn:infratographer:user" + u.ID,
		Source:     "identityapi.bootstrap",
		Timestamp:  time.Now(),
		SubjectFields: map[string]string{
			"id":         t.ID,
			"created_at": t.CreatedAt.Format(time.RFC3339),
			"updated_at": t.UpdatedAt.Format(time.RFC3339),
		},
	}); err != nil {
		logger.Fatalw("failed to publish event for tenant creation", "error", err)
	}

	if err := tx.Commit(); err != nil {
		logger.Fatalw("failed to commit DB transaction", "error", err)
	}

	fmt.Println("Bootstrap data successfully created.")
}

// askForConfirmation asks the user for confirmation. A user must type in "yes" or "no" and
// then press enter. It has fuzzy matching, so "y", "Y", "yes", "YES", and "Yes" all count as
// confirmations. If the input is not recognized, it will ask again. The function does not return
// until it gets a valid response from the user.
func askForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}
