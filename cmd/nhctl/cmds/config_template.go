package cmds

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"nocalhost/cmd/nhctl/cmds/tpl"
	"nocalhost/pkg/nhctl/log"
)

type CommonFlags struct {
	AppName string
	SvcName string
}

var commonFlags = CommonFlags{}

func init() {
	configTemplateCmd.Flags().StringVarP(&commonFlags.SvcName, "deployment", "d", "", "k8s deployment which your developing service exists")
	configCmd.AddCommand(configTemplateCmd)
}

var configTemplateCmd = &cobra.Command{
	Use:   "template [Name]",
	Short: "get service config template",
	Long:  "get service config template",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		commonFlags.AppName = args[0]
		InitAppAndCheckIfSvcExist(commonFlags.AppName, commonFlags.SvcName)
		t, err := tpl.GetSvcTpl(commonFlags.SvcName)
		if err != nil {
			log.Fatalf("fail to get svc tpl:%s", err.Error())
		}
		fmt.Println(t)
	},
}
