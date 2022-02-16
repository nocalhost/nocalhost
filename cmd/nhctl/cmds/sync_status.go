/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/retry"
	"nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/internal/nhctl/app"
	"nocalhost/internal/nhctl/common/base"
	"nocalhost/internal/nhctl/syncthing/network/req"
	"nocalhost/pkg/nhctl/log"
	"time"
)

var syncStatusOps = &app.SyncStatusOptions{}

func init() {
	//syncStatusCmd.Flags().StringVarP(&common.NameSpace, "namespace", "n", "", "kubernetes namespace")
	syncStatusCmd.Flags().StringVarP(
		&common.WorkloadName, "deployment", "d", string(base.Deployment),
		"k8s deployment which your developing service exists",
	)
	syncStatusCmd.Flags().StringVarP(
		&common.ServiceType, "controller-type", "t", "deployment",
		"kind of k8s controller,such as deployment,statefulSet",
	)
	syncStatusCmd.Flags().BoolVar(
		&syncStatusOps.Override, "override", false,
		"override the remote changing according to the local sync folder",
	)
	syncStatusCmd.Flags().BoolVar(
		&syncStatusOps.WaitForSync, "wait", false,
		"wait for first sync process finished, default value is false",
	)
	syncStatusCmd.Flags().BoolVar(
		&syncStatusOps.Watch, "watch", false,
		"wait for sync process finished, default value is false",
	)
	syncStatusCmd.Flags().Int64Var(
		&syncStatusOps.Timeout, "timeout", 120,
		"wait for sync process finished timeout, default is 120 seconds, unit is seconds ",
	)
	rootCmd.AddCommand(syncStatusCmd)
}

var syncStatusCmd = &cobra.Command{
	Use:   "sync-status [NAME]",
	Short: "Files sync status",
	Long:  "Tracing the files sync status, include local folder and remote device",
	Args: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			display(req.WelcomeTemplate)
			return
		}

		if data := SyncStatus(syncStatusOps, common.NameSpace, args[0], common.WorkloadName, common.ServiceType, common.KubeConfig); data != nil {
			display(data)
		}
	},
}

func SyncStatus(opt *app.SyncStatusOptions, ns, app, svc, svcType, kubeconfig string) *req.SyncthingStatus {
	common.NameSpace = ns
	common.KubeConfig = kubeconfig

	nocalhostApp, err := common.InitAppMutate(app)
	if err != nil {
		return req.AppNotInstalledTemplate
	}

	nhSvc, err := nocalhostApp.InitService(svc, svcType)
	if err != nil {
		return req.NotInDevModeTemplate
	}

	if !nhSvc.IsInDevMode() {
		return req.NotInDevModeTemplate
	}

	if nhSvc.IsInDevModeStarting() {
		return req.DevModeStarting
	}

	if !nhSvc.IsProcessor() {
		return req.NotProcessor
	}

	// check if syncthing exists
	//pid, err := nhSvc.GetSyncThingPid()
	//if err != nil {
	//	return req.NotSyncthingProcessFound
	//}
	//
	//pro, err := ps.FindProcess(pid)
	//if err != nil || pro == nil {
	//	return req.NotSyncthingProcessFound
	//}

	client := nhSvc.NewSyncthingHttpClient(2)

	if opt != nil {
		if opt.Override {
			must(client.FolderOverride())
			display("Succeed")
			return nil
		}

		if opt.WaitForSync {
			waitForFirstSync(client, time.Second*time.Duration(opt.Timeout))
			return nil
		}

		if opt.Watch {
			watchSyncProcess(client)
			return nil
		}
	}

	return client.GetSyncthingStatus()
}

func display(v interface{}) {
	marshal, _ := json.Marshal(v)
	fmt.Printf("%s", string(marshal))
}

func displayLn(v interface{}) {
	if v != nil {
		marshal, _ := json.Marshal(v)
		fmt.Println(string(marshal))
	}
}

func waitForFirstSync(client *req.SyncthingHttpClient, duration time.Duration) {
	ctx, cancelFunc := context.WithTimeout(context.Background(), duration)
	defer cancelFunc()

out:
	for {
		select {
		case <-ctx.Done():
			display(
				req.SyncthingStatus{Status: req.Error, Msg: "wait for sync connect timeout", Tips: "", OutOfSync: ""},
			)
			return
		default:
			time.Sleep(time.Second * 1)
			isConnected, err := client.SystemConnections()
			if err == nil && isConnected {
				break out
			}
		}
	}

	// scan folder
	err2 := retry.OnError(
		retry.DefaultBackoff, func(err error) bool {
			return err != nil
		}, func() error {
			return client.Scan()
		},
	)
	if err2 != nil {
		log.Logf("scan folder manually error, err: %v", err2)
	}

	for {
		select {
		case <-ctx.Done():
			display(
				req.SyncthingStatus{Status: req.Error, Msg: "wait for sync finished timeout", Tips: "", OutOfSync: ""},
			)
			return
		default:
			time.Sleep(time.Second * 1)
			if status := client.GetSyncthingStatus(); status != nil && status.Status == req.Idle {
				if events, err := client.Events(0); err == nil {
					for _, event := range events {
						if event.EventType == req.EventFolderCompletion && event.Data.Completion == 100 {
							display(req.SyncthingStatus{Status: req.Idle, Msg: "sync finished", Tips: "", OutOfSync: ""})
							return
						}
					}
				}
			}
		}
	}
}

func watchSyncProcess(client *req.SyncthingHttpClient) {
	ctx, cancelFunc := context.WithTimeout(context.TODO(), time.Hour*24)
	//ticker := time.NewTicker(time.Second * 1)
	defer cancelFunc()
	//defer ticker.Stop()
out:
	for {
		select {
		case <-ctx.Done():
			display(
				req.SyncthingStatus{Status: req.Error, Msg: "wait for sync connect timeout", Tips: "", OutOfSync: ""},
			)
			return
		default:
			time.Sleep(time.Second * 1)
			isConnected, err := client.SystemConnections()
			if err == nil && isConnected {
				break out
			}
		}
	}

	// get all events before scan
	lastId := int64(0)
	if events, err2 := client.Events(0); err2 == nil {
		lastId += int64(len(events))
	}
	// scan folder
	err2 := retry.OnError(
		retry.DefaultBackoff, func(err error) bool {
			return err != nil
		}, func() error {
			return client.Scan()
		},
	)
	if err2 != nil {
		log.Logf("scan folder manually error, err: %v", err2)
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			_ = retry.OnError(
				retry.DefaultBackoff, func(err error) bool {
					return err != nil
				}, func() error {
					return client.Scan()
				},
			)
			time.Sleep(time.Second * 2)
			found := false
			if status := client.GetSyncthingStatus(); status != nil && status.Status == req.Idle {
				if events, err := client.Events(lastId); err == nil {
					for _, event := range events {
						if event.EventType == req.EventFolderCompletion && event.Data.Completion == 100 {
							found = true
						}
						lastId = event.Id
					}
				}
			}
			if found {
				displayLn(req.SyncthingStatus{Status: req.Idle, Msg: "sync finished", Tips: "", OutOfSync: ""})
			}
		}
	}
}
