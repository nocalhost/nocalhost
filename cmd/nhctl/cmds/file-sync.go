package cmds

import (
	"fmt"
	"github.com/spf13/cobra"
	"nocalhost/pkg/nhctl/third_party/mutagen"
)

var localFolderName, remoteFolder, sshPort string

func init() {
	//install k8s
	//fileSyncCmd.Flags().StringVarP(&sessionName, "session", "s", "", "sync session")
	fileSyncCmd.Flags().StringVarP(&localFolderName, "local-folder", "l", "", "local folder path")
	fileSyncCmd.Flags().StringVarP(&remoteFolder, "remote-folder", "r", "/home/code", "remote folder path")
	fileSyncCmd.Flags().StringVarP(&sshPort, "port", "p", "22", "ssh port")
	//fileSyncCmd.Flags().StringVarP(&remoteFolder, "ssh passwd", "p", "", "ssh passwd")
	rootCmd.AddCommand(fileSyncCmd)
}

var fileSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync files to remote Pod in Kubernetes",
	Long: `Sync files to remote Pod in Kubernetes`,
	Run: func(cmd *cobra.Command, args []string) {
		if localFolderName == "" {
			fmt.Println("error: please use -l to specify a local directory to sync with remote")
			return
		}
		if remoteFolder == "" {
			fmt.Println("error: please use -r to specify a remote folder")
			return
		}
		//TO-DO
		fmt.Println("file syncing...")  // tools/darwin/mutagen sync create --sync-mode=one-way-safe --releaseName=$1  $2  $3
											// ./tools/script/file-sync.sh coding dir01 root@127.0.0.1:12345:/home/code
		mutagen.FileSync(localFolderName, remoteFolder, sshPort)
	},
}