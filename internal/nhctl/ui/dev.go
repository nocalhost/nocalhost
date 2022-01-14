/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"encoding/json"
	"fmt"
	"github.com/derailed/tview"
	"github.com/gdamore/tcell/v2"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	common2 "nocalhost/cmd/nhctl/cmds/common"
	"nocalhost/cmd/nhctl/cmds/dev"
	"nocalhost/cmd/nhctl/cmds/kube"
	"nocalhost/internal/nhctl/appmeta"
	_const "nocalhost/internal/nhctl/const"
	"nocalhost/internal/nhctl/daemon_client"
	"nocalhost/internal/nhctl/daemon_handler/item"
	"nocalhost/internal/nhctl/utils"
	yaml "nocalhost/pkg/nhctl/utils/custom_yaml_v3"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func (t *TviewApplication) buildSelectWorkloadList(appMeta *appmeta.ApplicationMeta, ns, wl string) *EnhancedTable {
	workloadListTable := t.NewBorderedTable(" Select a workload")

	workloadListTable.SetFocusFunc(func() {
		cli, err := daemon_client.GetDaemonClient(utils.IsSudoUser())
		if err != nil {
			t.showErr(err.Error())
		}

		data, err := cli.SendGetResourceInfoCommand(
			t.clusterInfo.KubeConfig, ns, appMeta.Application, wl,
			"", nil, false)

		bytes, err := json.Marshal(data)
		if err != nil {
			t.showErr(err.Error())
		}

		var items []item.Item
		if err = json.Unmarshal(bytes, &items); err != nil {
			t.showErr(err.Error())
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

			workloadListTable.SetCell(i+1, 2, coloredCell(""))
			workloadListTable.SetCell(i+1, 3, coloredCell("NONE"))
			workloadListTable.SetCell(i+1, 4, coloredCell("FALSE"))
			workloadListTable.SetCell(i+1, 5, coloredCell("0"))
			workloadListTable.SetCell(i+1, 6, coloredCell("[]"))

			name, ok, err := unstructured.NestedString(um, "metadata", "name")
			if err != nil || !ok {
				continue
			}
			workloadListTable.SetCell(i+1, 0, coloredCell(name))

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
	})

	var err error
	var selectWorkloadFunc = func(row, column int) {
		if row > 0 {
			workloadNameCell := workloadListTable.GetCell(row, 0)
			common2.WorkloadName = trimSpaceStr(workloadNameCell.Text)
			common2.ServiceType = "deployment"
			common2.KubeConfig = t.clusterInfo.KubeConfig
			common2.NameSpace = ns
			err = common2.InitAppAndCheckIfSvcExist(appMeta.Application, common2.WorkloadName, common2.ServiceType)
			if err != nil {
				t.showErr(err.Error())
				return
			}

			opsTable := NewBorderedTable("Menu")
			options := make([]string, 0)

			if common2.NocalhostSvc.IsInDevMode() {
				options = append(options, endDevModeOpt)
			} else {
				options = append(options, startDevModeOpt)
			}
			if !common2.NocalhostSvc.IsProcessor() {
				options = append(options, " Start DevMode(Duplicate)")
			}

			options = append(options, portForwardOpt, viewDevConfigOpt, " Reset Pod", viewLogsOpt, openTerminalOpt, syncLogsOpt, openGuiOpt)
			for i, option := range options {
				opsTable.SetCellSimple(i, 0, option)
			}

			x, y, _ := workloadNameCell.GetLastPosition()
			opsTable.SetRect(x+10, y, 30, 10)
			t.pages.AddPage("menu", opsTable, false, true)
			t.pages.ShowPage("menu")
			t.app.SetFocus(opsTable)
			opsTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
				if event.Key() == tcell.KeyEsc {
					t.pages.HidePage("menu")
					t.app.SetFocus(workloadListTable)
				}
				return event
			})
			opsTable.SetSelectedFunc(func(row1, column1 int) {
				t.pages.HidePage("menu")
				switch opsTable.GetCell(row1, column1).Text {
				case startDevModeOpt:
					containerTable := t.ShowContainersTable(x+10, y, 50, 10)
					if containerTable == nil {
						return
					}

					containerTable.SetSelectedFunc(func(row, column int) {
						t.pages.HidePage("containers")
						dev.DevStartOps.Container = trimSpaceStr(containerTable.GetCell(row, column).Text)
						svcConfig := common2.NocalhostSvc.Config()
						if svcConfig == nil {
							t.showErr("Svc config is nil")
							return
						}
						config := svcConfig.GetContainerDevConfigOrDefault(dev.DevStartOps.Container)
						configured := make(chan bool, 1)
						if config.Image == "" {
							imageTables := t.ShowDevImageTable(10, 10, 80, 20)
							imageTables.SetSelectedFunc(func(row, column int) {
								t.pages.HidePage(devImagesPage)
								image := imageTables.GetCell(row, 0).Text
								config.Image = image
								err = common2.NocalhostSvc.UpdateConfig(*svcConfig)
								if err != nil {
									t.showErr(err.Error())
									return
								}
								configured <- true
							})
						} else {
							configured <- true
						}

						go func() {
							select {
							case cf := <-configured:
								if !cf {
									return
								}
							}
							str, _ := os.Getwd()
							dev.DevStartOps.LocalSyncDir = []string{str}
							dev.DevStartOps.NoTerminal = true

							writer := t.switchBodyToScrollingView(" Start DevMode", workloadListTable)
							RedirectionOut(writer)
							go func() {
								dev.StartDevMode(appMeta.Application)
								RecoverOut()

								podList, err := common2.NocalhostSvc.GetPodList()
								if err != nil {
									t.showErr(err.Error())
									return
								}
								var runningPod = make([]v1.Pod, 0, 1)
								for _, item := range podList {
									if item.Status.Phase == v1.PodRunning && item.DeletionTimestamp == nil {
										runningPod = append(runningPod, item)
									}
								}
								if len(runningPod) != 1 {
									t.showErr(fmt.Sprintf("Pod num is %d (not 1), please specify one", len(runningPod)))
								}
								t.app.Suspend(func() {
									fmt.Print("\033[H\033[2J") // clear screen
									common2.NocalhostSvc.EnterPodTerminal(runningPod[0].Name, "nocalhost-dev", "bash", _const.DevModeTerminalBanner)
								})
								t.app.QueueUpdateDraw(func() {
									t.app.SetFocus(workloadListTable)
								})
							}()
						}()

					})
				case endDevModeOpt:
					writer := t.switchBodyToScrollingView(" End DevMode", workloadListTable)
					RedirectionOut(writer)
					go func() {
						dev.EndDevMode()
						RecoverOut()
					}()
				case portForwardOpt:
					i := tview.NewInputField()
					i.SetBorderPadding(0, 0, 1, 1)
					f := tview.NewFlex()
					f.SetDirection(tview.FlexRow)
					f.AddItem(i, 1, 1, true)
					f.SetRect(20, 5, 40, 20)
					//pfListTable := tview.NewTable()
					pfListTable := NewBorderedTable("")
					pfListTable.SetCellSimple(0, 0, "39080:9080")
					f.AddItem(pfListTable, 0, 1, true)
					f.SetBorder(true)
					t.pages.AddPage("pfPage", f, false, true)
					t.pages.ShowPage("pfPage")
					i.Autocomplete()
					i.SetDoneFunc(func(key tcell.Key) {
						if key == tcell.KeyEsc {
							t.pages.HidePage("pfPage")
						}
					})
				case viewDevConfigOpt:
					writer := t.switchBodyToScrollingView("Dev Config", workloadListTable)
					go func() {
						config := common2.NocalhostSvc.Config()
						bys, err := yaml.Marshal(config)
						if err != nil {
							t.showErr(err.Error())
							return
						}
						writer.Write(bys)
					}()
				case openTerminalOpt:
					go func() {
						podList, err := common2.NocalhostSvc.GetPodList()
						if err != nil {
							t.showErr(err.Error())
							return
						}
						var runningPod = make([]v1.Pod, 0, 1)
						for _, item := range podList {
							if item.Status.Phase == v1.PodRunning && item.DeletionTimestamp == nil {
								runningPod = append(runningPod, item)
							}
						}
						if len(runningPod) != 1 {
							t.showErr(fmt.Sprintf("Pod num is %d (not 1), please specify one", len(runningPod)))
						}
						t.app.Suspend(func() {
							fmt.Print("\033[H\033[2J") // clear screen
							common2.NocalhostSvc.EnterPodTerminal(runningPod[0].Name, "nocalhost-dev", "bash", _const.DevModeTerminalBanner)
						})
						t.app.QueueUpdateDraw(func() {
							t.app.SetFocus(workloadListTable)
						})
					}()
				case viewLogsOpt:
					containerTable := t.ShowContainersTable(x+10, y, 50, 10)
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
				case syncLogsOpt:
					logPath := filepath.Join(common2.NocalhostSvc.GetSyncDir(), "syncthing.log")
					if _, err = os.Stat(logPath); err != nil {
						t.showErr(err.Error())
						return
					}
					writer := t.switchBodyToScrollingView(" File Sync logs", workloadListTable)
					go func() {
						bys, err := ioutil.ReadFile(logPath)
						if err != nil {
							t.showErr(err.Error())
							return
						}
						if _, err := writer.Write(bys); err != nil {
							t.showErr(err.Error())
							return
						}
					}()
				case openGuiOpt:
					guiPortCell := workloadListTable.GetCell(row, 5)
					if guiPortCell == nil || guiPortCell.Text == "" || guiPortCell.Text == "0" {
						t.showErr("File Sync is not running")
						return
					}
					go func() {
						exec.Command(`open`, fmt.Sprintf("http://localhost:%s", guiPortCell.Text)).Start()
					}()
				}
			})
		}
	}
	workloadListTable.SetSelectedFunc(selectWorkloadFunc)
	workloadListTable.Select(1, 0)
	return workloadListTable
}

func (t *TviewApplication) ShowContainersTable(x, y, width, height int) *EnhancedTable {
	containerList, err := common2.NocalhostSvc.GetOriginalContainers()
	if err != nil {
		t.showErr(err.Error())
		return nil
	}
	containerTable := t.NewBorderedTable("Containers")
	for i, container := range containerList {
		containerTable.SetCellSimple(i, 0, " "+container.Name)
	}
	containerTable.SetRect(x, y, width, height)
	t.app.SetFocus(containerTable)
	t.pages.AddPage("containers", containerTable, false, true)
	t.pages.ShowPage("containers")
	containerTable.SetBlurFunc(func() {
		t.pages.HidePage("containers")
	})
	containerTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			t.pages.HidePage("containers")
			//t.app.SetFocus(workloadListTable)
		}
		return event
	})
	return containerTable
}

func (t *TviewApplication) ShowDevImageTable(x, y, width, height int) *EnhancedTable {

	devImages := []string{
		"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/java:11",
		"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/ruby:3.0",
		"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/node:14",
		"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/python:3.9",
		"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/golang:1.16",
		"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/perl:latest",
		"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/rust:latest",
		"nocalhost-docker.pkg.coding.net/nocalhost/dev-images/php:latest",
	}
	devImagesTable := t.NewBorderedTable("Dev Images")
	for i, devImage := range devImages {
		devImagesTable.SetCellSimple(i, 0, devImage)
	}
	devImagesTable.SetRect(x, y, width, height)
	t.app.SetFocus(devImagesTable)
	t.pages.AddPage(devImagesPage, devImagesTable, false, true)
	t.pages.ShowPage(devImagesPage)
	//devImagesTable.SetBlurFunc(func() {
	//	t.pages.HidePage(devImagesPage)
	//})
	devImagesTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			t.pages.HidePage(devImagesPage)
		}
		return event
	})
	return devImagesTable
}
