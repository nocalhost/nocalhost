package cmds

import (
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"nocalhost/pkg/nhctl/utils"
	"os"
)

var settings *EnvSettings

func init() {

	settings = NewEnvSettings()

	rootCmd.PersistentFlags().BoolVar(&settings.Debug, "debug", settings.Debug, "enable debug level log")

	cobra.OnInitialize(func() {
		var (
			nhctlHomeDirName = ".nhctl"
		)
		nhctlHomeDir := fmt.Sprintf("%s%c%s", GetHomePath(), os.PathSeparator, nhctlHomeDirName)
		if _, err := os.Stat(nhctlHomeDir); err != nil {
			if os.IsNotExist(err) {
				debug("init nhctl...")
				utils.Mush(os.Mkdir(nhctlHomeDir, 0755))
				keyDir := fmt.Sprintf("%s%c%s", nhctlHomeDir, os.PathSeparator, "key")
				utils.Mush(os.Mkdir(keyDir, 0755)) // create .nhctl/key
				// ssh public key
				keyContent := `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDqJOIfjQvv2pAanw3PBjpIqda+F7QAY0C4818D76C4u5Ybrja+Fz0cOCjtrRuwopsNcZhbGrva/zuG8J7Violft294fYVils7gOi1FjzA2twU1n90nCFpHt5uxETR9jR7JpsTUq15Xi6aIB5PynF/irr3EueUiiywhvzejbr1sA0ri26wteaSr/nLdNFy2TXVAEyHyzoxCAX4cECuGfarIgoQpdErc6dwyCh+lPnByL+AGP+PKsQmHmA/3NUUJGsurEf4vGaCd0d7/FGtvMG+N28C33Rv1nZi4RzWbG/TGlFleuvO8QV1zqIGQbUkqoeoLbbYsOW2GG0BxhJ7jqj9V root@eafa293b895`
				publicKeyFile := fmt.Sprintf("%s%c%s", keyDir, os.PathSeparator, "id_rsa.pub")
				utils.Mush(ioutil.WriteFile(publicKeyFile, []byte(keyContent), 0755))
			}
		}
	})

}

var rootCmd = &cobra.Command{
	Use:   "nhctl",
	Short: "nhctl use to deploy coding project",
	Long:  `nhctl can deploy project on Kubernetes. `,
	Run: func(cmd *cobra.Command, args []string) {
		debug("hello nhctl")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
