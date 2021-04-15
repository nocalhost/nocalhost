package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/syncthing/network/req"
)

var syncStatusOps = &app.SyncStatusOptions{}

func init() {
	//syncStatusCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	syncStatusCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	syncStatusCmd.Flags().BoolVar(&syncStatusOps.Override, "override", false, "override the remote changing according to the local sync folder")

	rootCmd.AddCommand(syncStatusCmd)
}

var syncStatusCmd = &cobra.Command{
	Use:   "sync-status [NAME]",
	Short: "Files sync status",
	Long:  "Tracing the files sync status, include local folder and remote device",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		initApp(applicationName)

		if b, _ := nocalhostApp.CheckIfSvcIsDeveloping(deployment); !b {
			display(req.NotInDevModeTemplate)
			return
		}

		client := nocalhostApp.NewSyncthingHttpClient(deployment)

		if syncStatusOps.Override {
			must(client.FolderOverride())
			display("Succeed")
			return
		}

		display(client.GetSyncthingStatus())
	},
}

func display(v interface{}) {
	marshal, _ := json.Marshal(v)
	fmt.Printf("%s", string(marshal))
}
