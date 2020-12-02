/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmds

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/syncthing"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
)

var fileSyncOps = &app.FileSyncOptions{}

func init() {
	fileSyncCmd.Flags().StringVarP(&fileSyncOps.LocalSharedFolder, "local-shared-folder", "l", "", "(Deprecation) local folder to sync")
	fileSyncCmd.Flags().StringVarP(&fileSyncOps.RemoteDir, "remote-folder", "r", "", "(Deprecation) remote folder path")
	fileSyncCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	fileSyncCmd.Flags().IntVarP(&fileSyncOps.LocalSshPort, "port", "p", 0, "(Deprecation) local port which forwards to remote ssh port")
	fileSyncCmd.Flags().BoolVarP(&fileSyncOps.RunAsDaemon, "daemon", "m", true, "if file sync run as daemon, default true")
	fileSyncCmd.Flags().BoolVarP(&fileSyncOps.SyncDouble, "double", "b", true, "if use double side sync, default true")
	rootCmd.AddCommand(fileSyncCmd)
}

var fileSyncCmd = &cobra.Command{
	Use:   "sync [NAME]",
	Short: "Sync files to remote Pod in Kubernetes",
	Long:  `Sync files to remote Pod in Kubernetes`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.Errorf("%q requires at least 1 argument\n", cmd.CommandPath())
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		applicationName := args[0]
		if !nh.CheckIfApplicationExist(applicationName) {
			log.Fatalf("application \"%s\" not found\n", applicationName)
		}
		nocalhostApp, err = app.NewApplication(applicationName)
		//clientgoutils.Must(err)
		if err != nil {
			log.Fatalf("fail to get app info\n")
		}
		if deployment == "" {
			// todo record default deployment
			log.Fatal("please use -d to specify a k8s deployment")
		}

		if !nocalhostApp.CheckIfSvcIsDeveloping(deployment) {
			log.Fatalf("\"%s\" has not in developing", deployment)
		}

		// get dev-start stage record free pod so it do not need get free port agian
		var devStartOptions = &app.DevStartOptions{}
		fileSyncOps, err = nocalhostApp.GetSyncthingPort(deployment, fileSyncOps)

		// syncthing port-forward
		// daemon
		// set Abs directory so can call myself
		NhctlAbsdir, err := exec.LookPath(nocalhostApp.GetMyBinName())
		if err != nil {
			log.Fatal("installing fortune is in your future")
		}
		// overwrite Args[0] as ABS directory of bin directory
		os.Args[0] = NhctlAbsdir

		// run in background
		if fileSyncOps.RunAsDaemon {
			// when child progress run here it will check env value and exit, so child will not run above code
			_, err := daemon.Background(nocalhostApp.GetPortSyncLogFile(deployment), nocalhostApp.GetApplicationBackGroundPortForwardPidFile(deployment), true)
			if err != nil {
				log.Fatalf("run port-forward background fail, please try again")
			}
			// success write pid file and exit father progress, stay child progress run
		}

		podName := ""
		podNameSpace := ""
		podsList, err := nocalhostApp.GetPodsFromDeployment(context.TODO(), nocalhostApp.AppProfile.Namespace, deployment)
		if err != nil {
			log.Fatalf(err.Error())
		}
		if podsList != nil {
			// get first pod
			podName = podsList.Items[0].Name
			podNameSpace = podsList.Items[0].Namespace
		}

		// start port-forward

		var wg sync.WaitGroup
		wg.Add(1)

		// stopCh control the port forwarding lifecycle. When it gets closed the
		// port forward will terminate
		stopCh := make(chan struct{}, 1)
		// readyCh communicate when the port forward is ready to get traffic
		readyCh := make(chan struct{})
		// stream is used to tell the port forwarder where to place its output or
		// where to expect input if needed. For the port forwarding we just need
		// the output eventually
		stream := genericclioptions.IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: os.Stderr,
		}

		// managing termination signal from the terminal. As you can see the stopCh
		// gets closed to gracefully handle its termination.

		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

		go func() {
			err := nocalhostApp.PortForwardAPod(clientgoutils.PortForwardAPodRequest{
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      podName,
						Namespace: podNameSpace,
					},
				},
				LocalPort: fileSyncOps.RemoteSyncthingPort,
				PodPort:   fileSyncOps.RemoteSyncthingPort,
				Streams:   stream,
				StopCh:    stopCh,
				ReadyCh:   readyCh,
			})
			if err != nil {
				log.Fatalf(err.Error())
			}
		}()

		select {
		// wait until port-forward success
		case <-readyCh:
			break
		}
		fmt.Printf("Port forwarding is ready to get traffic!\n")

		// create new syncthing
		// TODO
		// If the file is deleted remotely, but the syncthing database is not reset (the development is not finished), the files that have been synchronized will not be synchronized.
		devStartOptions, err = nocalhostApp.GetSyncthingLocalDirFromProfileSaveByDevStart(deployment, devStartOptions)
		newSyncthing, err := syncthing.New(nocalhostApp, deployment, devStartOptions, fileSyncOps)
		// starts up a local syncthing
		err = newSyncthing.Run(context.TODO())

		// set sync status in child progress
		err = nocalhostApp.SetSyncingStatus(deployment, true)
		if err != nil {
			log.Fatalf("[error] fail to update syncing status")
		}

		for {
			<-sigs
			fmt.Println("stop port forward")
			close(stopCh)
			wg.Done()
		}
	},
}
