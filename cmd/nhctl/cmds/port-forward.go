package cmds

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"nocalhost/pkg/nhctl/third_party/kubectl"
	"os"
	"os/signal"
	"syscall"
)

var deployment, localPort, remotePort string

func init() {
	portForwardCmd.Flags().StringVarP(&nameSpace, "namespace", "n", "", "kubernetes namespace")
	//portForwardCmd.Flags().StringVarP(&kubeconfig, "kubeconfig", "k", "", "kubernetes cluster config")
	portForwardCmd.Flags().StringVarP(&localPort, "local-port", "l", "10000", "local port to forward")
	portForwardCmd.Flags().StringVarP(&remotePort, "remote-port", "r", "22", "remote port to be forwarded")
	portForwardCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which you want to forward to")
	rootCmd.AddCommand(portForwardCmd)
}

var portForwardCmd = &cobra.Command{
	Use:   "port-forward",
	Short: "Forward local port to remote pod'port",
	Long:  `Forward local port to remote pod'port`,
	Run: func(cmd *cobra.Command, args []string) {
		if deployment == "" {
			fmt.Println("error: please use -d to specify a kubernetes deployment")
			return
		}
		if nameSpace == "" {
			fmt.Println("error: please use -n to specify a kubernetes namespace")
			return
		}
		// todo local port should be specificed ?
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGINT) // kill -1
		ctx, cancel := context.WithCancel(context.TODO())

		go func() {
			<-c
			cancel()
			fmt.Println("stop port forward")
			CleanupPid()
		}()

		// check if there is a active port-forward
		_, err := os.Stat(".pid")
		if err == nil {
			fmt.Println("a port-forward process already existed")
			return
		} else {
			// record pid
			fmt.Println("recording pid...")
			pid := os.Getpid()
			ioutil.WriteFile(".pid", []byte(fmt.Sprintf("%d", pid)), 0644)
		}
		err = kubectl.PortForward(ctx, settings.KubeConfig, nameSpace, deployment, localPort, remotePort) // eg : ./utils/darwin/kubectl port-forward --address 0.0.0.0 deployment/coding  12345:22
		if err != nil {
			fmt.Printf("failed to forward port : %v\n", err)
			CleanupPid()
		}
	},
}

func CleanupPid() {
	if _, err2 := os.Stat(".pid"); err2 != nil {
		if os.IsNotExist(err2) {
			return
		}
	}
	err := os.Remove(".pid")
	if err != nil {
		fmt.Printf("removing .pid failed, please remove it manually, err:%v\n", err)
	} else {
		fmt.Println(".pid removed.")
	}
}
