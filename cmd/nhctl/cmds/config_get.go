package cmds

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"nocalhost/pkg/nhctl/log"
)

func init() {
	configGetCmd.Flags().StringVarP(&commonFlags.SvcName, "deployment", "d", "", "k8s deployment which your developing service exists")
	configCmd.AddCommand(configGetCmd)
}

var configGetCmd = &cobra.Command{
	Use:   "get [Name]",
	Short: "get application/service config",
	Long:  "get application/service config",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		commonFlags.AppName = args[0]
		InitApp(commonFlags.AppName)

		if commonFlags.SvcName == "" {
			if nocalhostApp.Config != nil {
				bys, err := yaml.Marshal(nocalhostApp.Config)
				if err != nil {
					log.Fatalf("fail to get application config: %s", err.Error())
				}
				fmt.Println(string(bys))
			}
		} else {
			CheckIfSvcExist(commonFlags.SvcName)
			svcConfig := nocalhostApp.GetSvcConfig(commonFlags.SvcName)
			if svcConfig != nil {
				bys, err := yaml.Marshal(svcConfig)
				if err != nil {
					log.Fatalf("fail to get svc config: %s", err.Error())
				}
				fmt.Println(string(bys))
			}
		}
	},
}
