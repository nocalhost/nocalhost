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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/syncthing/daemon"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var fileSyncOps = &app.FileSyncOptions{}

func init() {
	fileSyncCmd.Flags().StringVarP(&deployment, "deployment", "d", "", "k8s deployment which your developing service exists")
	fileSyncCmd.Flags().BoolVarP(&fileSyncOps.RunAsDaemon, "daemon", "m", true, "if file sync run as daemon")
	fileSyncCmd.Flags().BoolVarP(&fileSyncOps.SyncDouble, "double", "b", false, "if use double side sync")
	fileSyncCmd.Flags().StringSliceVarP(&fileSyncOps.SyncedPattern, "synced-pattern", "s", []string{}, "local synced pattern")
	fileSyncCmd.Flags().StringSliceVarP(&fileSyncOps.IgnoredPattern, "ignored-pattern", "i", []string{}, "local ignored pattern")
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

		InitAppAndCheckIfSvcExist(applicationName, deployment)

		if !nocalhostApp.CheckIfSvcIsDeveloping(deployment) {
			log.Fatalf("Service \"%s\" is not in developing", deployment)
		}

		if nocalhostApp.CheckIfSvcIsSyncthing(deployment) {
			log.Fatalf("Service \"%s\" is already in syncing", deployment)
		}

		// get dev-start stage record free pod so it do not need get free port again
		//var devStartOptions = &app.DevStartOptions{}

		// syncthing port-forward
		// daemon
		// set abs directory to call myself
		nhctlAbsDir, err := exec.LookPath(nocalhostApp.GetMyBinName())
		if err != nil {
			log.Fatal("installing fortune is in your future")
		}

		// Reconfirm whether devcontainer is ready
		pod := ""
		//for {
		<-time.NewTimer(time.Second * 1).C
		pod, err = nocalhostApp.WaitAndGetNocalhostDevContainerPod(deployment)
		if err != nil {
			log.Fatal(err)
		}
		//else {
		//break
		//}
		//log.Infof("wait for sidecar ready")
		//}

		log.Infof("Syncthing port-forward pod %s, namespace %s", pod, nocalhostApp.GetNamespace())

		// overwrite Args[0] as ABS directory of bin directory
		os.Args[0] = nhctlAbsDir

		// run in background
		if fileSyncOps.RunAsDaemon {
			// when child progress run here, it will check env value and exit, so child will not run above code
			_, err := daemon.Background(nocalhostApp.GetPortSyncLogFile(deployment), nocalhostApp.GetApplicationBackGroundPortForwardPidFile(deployment), true)
			if err != nil {
				log.Fatalf("run port-forward background fail, please try again")
			}
			// success write pid file and exit father progress, stay child progress run
		}

		podName, err := nocalhostApp.WaitAndGetNocalhostDevContainerPod(deployment)
		if err != nil {
			log.Fatalf(err.Error())
		}

		log.Infof("Syncthing port-forward pod %s, namespace %s", podName, nocalhostApp.GetNamespace())

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

		svcProfile := nocalhostApp.GetSvcProfile(deployment)
		go func() {
			err := nocalhostApp.PortForwardAPod(clientgoutils.PortForwardAPodRequest{
				Pod: v1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      podName,
						Namespace: nocalhostApp.GetNamespace(),
					},
				},
				LocalPort: svcProfile.RemoteSyncthingPort,
				PodPort:   svcProfile.RemoteSyncthingPort,
				Streams:   stream,
				StopCh:    stopCh,
				ReadyCh:   readyCh,
			})
			if err != nil {
				log.Fatal(err.Error())
			}
		}()

		select {
		// wait until port-forward success
		case <-readyCh:
			break
		}
		log.Info("Port forwarding is ready to get traffic!")

		// Deprecated for multi dir sync
		// On latest version, it only use for specify the sync dir
		//devStartOptions, err = nocalhostApp.GetSyncthingLocalDirFromProfileSaveByDevStart(deployment, devStartOptions)
		//if err != nil {
		//	log.Fatalf("failed to get syncthing local dir")
		//}

		// Getting pattern from svc profile first
		profile := nocalhostApp.GetSvcProfile(deployment)
		if len(fileSyncOps.IgnoredPattern) != 0 {
			profile.IgnoredPattern = fileSyncOps.IgnoredPattern
		}
		if len(fileSyncOps.SyncedPattern) != 0 {
			profile.SyncedPattern = fileSyncOps.SyncedPattern
		}

		// TODO
		// If the file is deleted remotely, but the syncthing database is not reset (the development is not finished), the files that have been synchronized will not be synchronized.
		newSyncthing, err := nocalhostApp.NewSyncthing(deployment, profile.LocalAbsoluteSyncDirFromDevStartPlugin, fileSyncOps.SyncDouble)
		if err != nil {
			log.WarnE(err, "Failed to new syncthing")
		}

		// starts up a local syncthing
		err = newSyncthing.Run(context.TODO())
		if err != nil {
			log.WarnE(err, "Failed to run syncthing")
		}

		// set sync status in child progress
		err = nocalhostApp.SetSyncingStatus(deployment, true)
		if err != nil {
			log.Fatal("Failed to update syncing status")
		}

		for {
			<-sigs
			log.Info("Stopping port forward")
			close(stopCh)
			wg.Done()
		}
	},
}
