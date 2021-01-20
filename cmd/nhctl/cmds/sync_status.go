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
	syncStatusCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	syncStatusCmd.Flags().BoolVar(&syncStatusOps.Override, "override", false, "override the remote changing according to the local sync folder")

	rootCmd.AddCommand(syncStatusCmd)
}

var syncStatusCmd = &cobra.Command{
	Use:   "sync-status [NAME]",
	Short: "TODO",
	Long:  "TODO",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		applicationName := args[0]
		InitAppAndCheckIfSvcExist(applicationName, deployment)

		if !nocalhostApp.CheckIfSvcIsDeveloping(deployment) {
			display(req.NotInDevModeTemplate)
			return
		}

		if nocalhostApp.CheckIfSvcIsSyncthing(deployment) {
			fmt.Print("zzzzzzzzzzz")
			display(req.FileSyncNotRunningTemplate)
			return
		}
		display(nocalhostApp.NewSyncthingHttpClient(deployment).GetSyncthingStatus())
	},
}

func display(syncStatus *req.SyncthingStatus) {
	marshal, _ := json.Marshal(syncStatus)
	fmt.Printf("%s", string(marshal))
}
