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
	"go.infratographer.com/tenant-api/internal/ent/generated/tenant"
)

var tenantList = &cobra.Command{
	Use:   "list",
	Short: "List Tenants",
	Run:   listTenant,
}

func init() {
	tenantCmd.AddCommand(tenantList)

	events.MustViperFlagsForPublisher(viper.GetViper(), tenantList.Flags(), appName)
	permissions.MustViperFlags(viper.GetViper(), tenantList.Flags())

	tenantList.Flags().Bool("all", false, "query all")
	tenantList.Flags().String("only", "", "only get the provided tenant id")
	tenantList.Flags().String("parent", "", "parent tenant id")
}

func listTenant(cmd *cobra.Command, _ []string) {
	client, closeFn := initializeGraphClient()
	defer closeFn()

	query := client.Tenant.Query()

	var (
		tenants []*ent.Tenant
		err     error
	)

	if all, _ := cmd.Flags().GetBool("all"); all {
		tenants, err = query.All(cmd.Context())
		if err != nil {
			logger.Fatalw("failed to query all tenants", "error", err)
		}
	} else if only, _ := cmd.Flags().GetString("only"); only != "" {
		onlyID, _ := gidx.Parse(only)
		tenants, err = query.Where(tenant.IDEQ(onlyID)).All(cmd.Context())
		if err != nil {
			logger.Fatalw("failed to get tenant", "error", err)
		}
	} else if parent, _ := cmd.Flags().GetString("parent"); parent != "" {
		parentID, _ := gidx.Parse(parent)
		tenants, err = query.Where(tenant.ParentTenantIDEQ(parentID)).All(cmd.Context())
		if err != nil {
			logger.Fatalw("failed to query all children", "error", err)
		}
	} else {
		tenants, err = query.Where(tenant.ParentTenantIDIsNil()).All(cmd.Context())
		if err != nil {
			logger.Fatalw("failed to query all root tenants", "error", err)
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")

	if err := enc.Encode(tenants); err != nil {
		logger.Fatalw("failed to encode payload", "error", err)
	}
}
