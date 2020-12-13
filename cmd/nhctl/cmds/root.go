package cmds

import (
	"fmt"
	"os"

	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
	"nocalhost/pkg/nhctl/utils"

	"github.com/spf13/cobra"
)

var settings *EnvSettings
var nh *nocalhost.NocalHost
var nocalhostApp *app.Application

func init() {

	settings = NewEnvSettings()

	rootCmd.PersistentFlags().BoolVar(&settings.Debug, "debug", settings.Debug, "enable debug level log")
	rootCmd.PersistentFlags().StringVar(&settings.KubeConfig, "kubeconfig", "", "the path of the kubeconfig file")

	cobra.OnInitialize(func() {
		var err error
		nh, err = nocalhost.NewNocalHost()
		utils.Mush(err)
	})
}

var rootCmd = &cobra.Command{
	Use:   "nhctl",
	Short: "nhctl use to deploy coding project",
	Long:  `nhctl can deploy and develop application on Kubernetes. `,
	Run: func(cmd *cobra.Command, args []string) {
		log.Debug("hello nhctl")
		//fmt.Printf("kubeconfig is %s", settings.KubeConfig)
	},
}

func Execute() {

	if len(os.Args) == 1 {
		args := append([]string{"help"}, os.Args[1:]...)
		rootCmd.SetArgs(args)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
