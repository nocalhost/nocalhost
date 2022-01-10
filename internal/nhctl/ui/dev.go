/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"encoding/json"
	"fmt"
	"github.com/derailed/tview"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	common2 "nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/cmd/nhctl/cmds/dev"
	"nocalhost/cmd/nhctl/cmds/kube"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_handler/item"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func (t *TviewApplication) buildSelectWorkloadList(appMeta *appmeta.ApplicationMeta) *tview.Table {
	workloadListTable := NewBorderedTable(" Select a workload")

	cli, err := daemon_client.GetDaemonClient(utils.IsSudoUser())
	if err != nil {
		showErr(err)
		return nil
	}

	data, err := cli.SendGetResourceInfoCommand(
		t.clusterInfo.KubeConfig, t.clusterInfo.NameSpace, appMeta.Application, "deployment",
		"", nil, false)

	bytes, err := json.Marshal(data)
	if err != nil {
		showErr(err)
		return nil
	}

	multiple := reflect.ValueOf(data).Kind() == reflect.Slice
	var items []item.Item
	var item2 item.Item
	if multiple {
		_ = json.Unmarshal(bytes, &items)
	} else {
		_ = json.Unmarshal(bytes, &item2)
	}
	if !multiple {
		items = append(items, item2)
	}

	for col, section := range []string{" Name", "Kind", "DevMode", "DevStatus", "Syncing", "SyncGUIPort", "PortForward"} {
		workloadListTable.SetCell(0, col, infoCell(section))
	}

	for i, it := range items {
		workloadListTable.SetCell(i+1, 0, coloredCell(" "+"name"))
		um, ok := it.Metadata.(map[string]interface{})
		if !ok {
			continue
		}

		name, ok, err := unstructured.NestedString(um, "metadata", "name")
		if err != nil || !ok {
			continue
		}
		workloadListTable.SetCell(i+1, 0, coloredCell(" "+name))

		kind, _, _ := unstructured.NestedString(um, "kind")
		if kind != "" {
			workloadListTable.SetCell(i+1, 1, coloredCell(kind))
		}

		if it.Description == nil {
			continue
		}

		workloadListTable.SetCell(i+1, 2, coloredCell(strings.ToUpper(string(it.Description.DevModeType))))
		workloadListTable.SetCell(i+1, 3, coloredCell(strings.ToUpper(it.Description.DevelopStatus)))
		workloadListTable.SetCell(i+1, 4, coloredCell(strings.ToUpper(strconv.FormatBool(it.Description.Syncing))))
		workloadListTable.SetCell(i+1, 5, coloredCell(strconv.Itoa(it.Description.LocalSyncthingGUIPort)))
		pfList := make([]string, 0)
		for _, forward := range it.Description.DevPortForwardList {
			pfList = append(pfList, fmt.Sprintf("%d->%d", forward.LocalPort, forward.RemotePort))
		}
		workloadListTable.SetCell(i+1, 6, coloredCell(fmt.Sprintf("%v", pfList)))
	}

	var selectWorkloadFunc = func(row, column int) {
		if row > 0 {
			table := NewBorderedTable("Menu")
			table.SetCellSimple(0, 0, startDevModeOpt)
			table.SetCellSimple(1, 0, " Start DevMode(Duplicate)")
			table.SetCellSimple(2, 0, " Port Forward")
			table.SetCellSimple(3, 0, " Reset Pod")
			table.SetCellSimple(4, 0, viewLogsOpt)
			table.SetCellSimple(5, 0, " Open Terminal")
			table.SetRect(20, 10, 30, 10)
			t.pages.AddPage("menu", table, false, true)
			t.pages.ShowPage("menu")

			t.app.SetFocus(table)
			table.SetSelectedFunc(func(row1, column1 int) {
				switch table.GetCell(row1, column1).Text {
				case startDevModeOpt:
					t.pages.HidePage("menu")
					workloadNameCell := workloadListTable.GetCell(row, 0)
					common2.WorkloadName = trimSpaceStr(workloadNameCell.Text)
					common2.ServiceType = "deployment"
					common2.KubeConfig = t.clusterInfo.KubeConfig
					common2.NameSpace = t.clusterInfo.NameSpace
					common2.InitAppAndCheckIfSvcExist(appMeta.Application, common2.WorkloadName, common2.ServiceType)

					containerList, err := common2.NocalhostSvc.GetOriginalContainers()
					if err != nil {
						t.showErr(err.Error())
						return
					}
					containerTable := NewBorderedTable("Containers")
					for i, container := range containerList {
						containerTable.SetCellSimple(i, 0, " "+container.Name)
					}
					containerTable.SetRect(20, 10, 50, 10)
					t.app.SetFocus(containerTable)
					t.pages.AddPage("containers", containerTable, false, true)
					t.pages.ShowPage("containers")
					containerTable.SetSelectedFunc(func(row, column int) {
						t.pages.HidePage("containers")
						dev.DevStartOps.Container = trimSpaceStr(containerTable.GetCell(row, column).Text)
						str, _ := os.Getwd()
						dev.DevStartOps.LocalSyncDir = []string{str}

						view := NewScrollingTextView(" Start DevMode")

						sbd := SyncBuilder{func(p []byte) (int, error) {
							t.app.QueueUpdateDraw(func() {
								view.Write([]byte(" " + string(p)))
							})
							return 0, nil
						}}

						log.RedirectionDefaultLogger(&sbd)
						go func() {
							dev.StartDevMode(appMeta.Application)
							log.RedirectionDefaultLogger(os.Stdout)
						}()
						t.switchBodyToC(workloadListTable, view)
					})
				case viewLogsOpt:
					t.pages.HidePage("menu")
					workloadNameCell := workloadListTable.GetCell(row, 0)
					//t.showErr(workloadNameCell.Text)
					common2.WorkloadName = trimSpaceStr(workloadNameCell.Text)
					common2.ServiceType = "deployment"
					common2.KubeConfig = t.clusterInfo.KubeConfig
					common2.NameSpace = t.clusterInfo.NameSpace
					err = common2.InitAppAndCheckIfSvcExist(appMeta.Application, common2.WorkloadName, common2.ServiceType)
					if err != nil {
						t.showErr(err.Error())
						return
					}

					containerList, err := common2.NocalhostSvc.GetOriginalContainers()
					if err != nil {
						t.showErr(err.Error())
						return
					}
					containerTable := NewBorderedTable("Containers")
					for i, container := range containerList {
						containerTable.SetCellSimple(i, 0, " "+container.Name)
					}
					containerTable.SetRect(20, 10, 50, 10)
					t.app.SetFocus(containerTable)
					t.pages.AddPage("containers", containerTable, false, true)
					t.pages.ShowPage("containers")
					containerTable.SetSelectedFunc(func(row, column int) {
						t.pages.HidePage("containers")
						writer := t.switchBodyToScrollingView(" View logs", workloadListTable)
						kube.InitLogOptions()
						kube.LogOptions.IOStreams.Out = writer
						// nhctl k logs --tail=3000 reviews-6446f84d54-8hdrt --namespace nocalhost-aaaa -c reviews --kubeconfig xxx
						kube.LogOptions.Container = trimSpaceStr(containerTable.GetCell(row, column).Text)
						kube.LogOptions.Tail = 3000
						// pod name?
						ps, err := common2.NocalhostSvc.GetPodList()
						if err != nil {
							t.showErr(err.Error())
							return
						}
						// todo: support multi pod
						if len(ps) != 1 {
							t.showErr("Pod len is not 1")
							return
						}
						go func(podName string) {
							// todo: we may need to recreate LogOptions
							kube.RunLogs(kube.CmdLogs, []string{podName})
						}(ps[0].Name)
					})

				}

			})
		}
	}
	workloadListTable.SetSelectedFunc(selectWorkloadFunc)
	workloadListTable.Select(1, 0)
	return workloadListTable
}
