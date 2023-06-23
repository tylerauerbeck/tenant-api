package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.infratographer.com/x/events"
	"go.infratographer.com/x/gidx"

	"go.infratographer.com/permissions-api/pkg/permissions"

	ent "go.infratographer.com/tenant-api/internal/ent/generated"
)

var tenantCreateCmd = &cobra.Command{
	Use:   "create NAME",
	Short: "Create a tenant",
	Args:  cobra.MinimumNArgs(1),
	Run:   createTenant,
}

func init() {
	tenantCmd.AddCommand(tenantCreateCmd)

	events.MustViperFlagsForPublisher(viper.GetViper(), tenantCreateCmd.Flags(), appName)
	permissions.MustViperFlags(viper.GetViper(), tenantCreateCmd.Flags())

	tenantCreateCmd.Flags().String("description", "", "description of tenant")
	tenantCreateCmd.Flags().String("parent", "", "parent tenant id")
}

func createTenant(cmd *cobra.Command, args []string) {
	client, closeFn := initializeGraphClient()
	defer closeFn()

	tenantName := args[0]

	var tenantDescription *string

	description, err := cmd.Flags().GetString("description")
	if err != nil {
		logger.Fatalw("failed to get description flag value", "error", err)
	}

	if description != "" {
		tenantDescription = &description
	}

	parent, err := cmd.Flags().GetString("parent")
	if err != nil {
		logger.Fatalw("failed to get parent flag value", "error", err)
	}

	var tenantParentID *gidx.PrefixedID

	if parent != "" {
		parentID, err := gidx.Parse(parent)
		if err != nil {
			logger.Fatalw("failed to parse parent ID", "error", err)
		}

		tenantParentID = &parentID
	}

	tenant, err := client.Tenant.Create().SetInput(
		ent.CreateTenantInput{
			Name:        tenantName,
			Description: tenantDescription,
			ParentID:    tenantParentID,
		},
	).Save(cmd.Context())
	if err != nil {
		logger.Fatalw("failed to create tenant", "error", err)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	if err := enc.Encode(tenant); err != nil {
		logger.Fatalw("failed to encode payload", "error", err)
	}
}
