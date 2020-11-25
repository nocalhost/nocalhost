package cmds

import (
	"fmt"
	"github.com/spf13/cobra"
	"nocalhost/internal/nhctl/app"
	nocalhost2 "nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/utils"
	"os"
)

var settings *EnvSettings
var nh *nocalhost2.NocalHost
var nocalhostApp *app.Application

func init() {

	settings = NewEnvSettings()

	rootCmd.PersistentFlags().BoolVar(&settings.Debug, "debug", settings.Debug, "enable debug level log")
	rootCmd.PersistentFlags().StringVar(&settings.KubeConfig, "kubeconfig", "", "the path to the kubeconfig file")

	cobra.OnInitialize(func() {
		var err error
		nh, err = nocalhost2.NewNocalHost()
		utils.Mush(err)
	})
}

var rootCmd = &cobra.Command{
	Use:   "nhctl",
	Short: "nhctl use to deploy coding project",
	Long:  `nhctl can deploy project on Kubernetes. `,
	Run: func(cmd *cobra.Command, args []string) {
		debug("hello nhctl")
		//fmt.Printf("kubeconfig is %s", settings.KubeConfig)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
