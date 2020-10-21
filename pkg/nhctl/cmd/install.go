package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"nocalhost/pkg/nhctl/tools"
	"strings"
)

var releaseName, gitUrl, helmDir, helmValueFile string

func init() {
	installCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	installCmd.Flags().StringVarP(&releaseName, "release-releaseName", "r", "", "release releaseName of helm")
	installCmd.Flags().StringVarP(&gitUrl, "git-url", "u", "", "url of git")
	installCmd.Flags().StringVarP(&helmDir, "dir", "d", "", "the dir of helm package")
	installCmd.Flags().StringVarP(&helmValueFile, "", "f", "", "helm's Value.yaml")
	installCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "kubernetes cluster config")
	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "install k8s application",
	Long:  `install k8s application`,
	Run: func(cmd *cobra.Command, args []string) {
		if nameSpace == "" {
			fmt.Println("error: please use -n to specify a kubernetes namespace")
			return
		}
		if releaseName == "" {
			fmt.Println("error: please use -r to specify the release name of helm")
			return
		}
		//if helmDir == "" {
		//	fmt.Println("error: please use -d to specify the dir of helm package")
		//	return
		//}
		if gitUrl == "" {
			fmt.Println("error: please use -u to specify url of git")
			return
		}
		fmt.Println("install helm application...")
		InstallHelmApplication()
	},
}


func InstallHelmApplication(){

	// clone git
	_, err := tools.ExecCommand(nil,true,"git", "clone", gitUrl)
	if err != nil {
		printlnErr("fail to clone git", err)
		return
	}

	// helm install
	gitSuffix := strings.TrimRight(gitUrl, ".git")
	strs := strings.Split(gitSuffix, "/")
	gitSuffix = strs[len(strs) - 1]
	chartPath := gitSuffix + "/"
	if helmDir != ""{
		chartPath += helmDir + "/"
	}
	fmt.Printf("chart path is %s\n", chartPath)
	_, err = tools.ExecCommand(nil,true,"helm", "upgrade", "--install", "--wait", releaseName, chartPath, "-n", nameSpace, "--kubeconfig", kubeconfig)
	if err != nil {
		printlnErr("fail to install helm app", err)
		return
	}

}