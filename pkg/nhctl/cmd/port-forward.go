package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"nocalhost/pkg/nhctl/third_party/kubectl"
	"os"
	"os/signal"
	"syscall"
)

var deployment,localPort,remotePort string

func init() {
	portForwardCmd.Flags().StringVarP(&localPort, "local", "l", "10000", "local port to forward")
	portForwardCmd.Flags().StringVarP(&remotePort, "remote-port", "r", "22", "remote port to be forwarded")
	portForwardCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "the k8s deployment we want to forward to")
	rootCmd.AddCommand(portForwardCmd)
}

var portForwardCmd = &cobra.Command{
	Use:   "port-forward",
	Short: "Forward local port to remote pod'port",
	Long: `Forward local port to remote pod'port`,
	Run: func(cmd *cobra.Command, args []string) {
		if deployment == "" {
			fmt.Println("error: deployment can not be empty")
			return
		}
		// todo local port should be specificed ?
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGHUP)  // kill -1
		ctx, cancel := context.WithCancel(context.TODO())

		go func() {
			<-c
			cancel()
			fmt.Println("stop port forward")
		}()

		kubectl.PortForward(ctx , deployment, localPort, remotePort) // eg : ./utils/darwin/kubectl port-forward --address 0.0.0.0 deployment/coding  12345:22
	},
}
