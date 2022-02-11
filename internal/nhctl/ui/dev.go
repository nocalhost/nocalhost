/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"encoding/json"
	"errors"
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
	"nocalhost/internal/nhctl/model"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/internal/nhctl/utils"
	yaml "nocalhost/pkg/nhctl/utils/custom_yaml_v3"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func (t *TviewApplication) buildWorkloadList(appMeta *appmeta.ApplicationMeta, ns, wl string) *EnhancedTable {
	workloadListTable := t.NewBorderedTable("")

	workloadListTable.SetFocusFunc(func() {
		cli, err := daemon_client.GetDaemonClient(utils.IsSudoUser())
		if err != nil {
			t.showErr(err, nil)
			return
		}

		data, err := cli.SendGetResourceInfoCommand(
			t.clusterInfo.KubeConfig, ns, appMeta.Application, wl,
			"", nil, false)

		bytes, err := json.Marshal(data)
		if err != nil {
			t.showErr(err, nil)
			return
		}

		var items []item.Item
		if err = json.Unmarshal(bytes, &items); err != nil {
			t.showErr(err, nil)
			return
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

	var selectWorkloadFunc = func(row, column int) {
		if row > 0 {
			workloadNameCell := workloadListTable.GetCell(row, 0)
			common2.WorkloadName = trimSpaceStr(workloadNameCell.Text)
			common2.ServiceType = strings.ToLower(strings.TrimSuffix(wl, "s"))
			common2.KubeConfig = t.clusterInfo.KubeConfig
			common2.NameSpace = ns
			nocalhostApp, nocalhostSvc, err := common2.InitAppAndCheckIfSvcExist(appMeta.Application, common2.WorkloadName, common2.ServiceType)
			if err != nil {
				t.showErr(err, nil)
				return
			}

			getPodNameList := func() ([]string, error) {
				pl, err := nocalhostSvc.GetPodList()
				if err != nil {
					return nil, err
				}
				result := make([]string, 0)
				for _, pod := range pl {
					result = append(result, pod.Name)
				}
				return result, nil
			}

			opsTable := NewRowSelectableTable("")
			options := make([]string, 0)

			if nocalhostSvc.IsInDevMode() {
				options = append(options, endDevModeOpt)
			} else {
				options = append(options, startDevModeOpt)
			}
			if !nocalhostSvc.IsProcessor() {
				options = append(options, startDupDevModeOpt)
			}

			options = append(options, portForwardOpt, viewDevConfigOpt, "Reset Pod", viewLogsOpt, openTerminalOpt, syncLogsOpt, openGuiOpt,
				viewProfile, viewDBData)
			for i, option := range options {
				opsTable.SetCell(i, 0, tview.NewTableCell(option).SetTextColor(tcell.Color(4294967449)))
			}

			x, y, _ := workloadNameCell.GetLastPosition()
			opsTable.SetRect(x+10, y, 30, len(options)+2)
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
				ops := opsTable.GetCell(row1, column1).Text
				switch ops {
				case startDevModeOpt, startDupDevModeOpt:
					devStartOps := &model.DevStartOptions{}
					containerList, err := nocalhostSvc.GetOriginalContainers()
					if err != nil {
						t.showErr(err, nil)
						return
					}
					containerTable := t.containersTable(containerList, x+10, y, 50, 10)
					if containerTable == nil {
						return
					}

					containerTable.SetSelectedFunc(func(row, column int) {
						t.pages.HidePage("containers")
						devStartOps.Container = containerTable.GetCell(row, column).Text
						svcConfig := nocalhostSvc.Config()
						if svcConfig == nil {
							t.showErr(errors.New("svc config is nil"), nil)
							return
						}
						config := svcConfig.GetContainerDevConfigOrDefault(devStartOps.Container)
						configured := make(chan bool, 1)
						if config.Image == "" {
							imageTables := t.ShowDevImageTable(10, 10, 80, 20)
							imageTables.SetSelectedFunc(func(row, column int) {
								t.pages.HidePage(devImagesPage)
								image := imageTables.GetCell(row, 0).Text
								config.Image = image
								err = nocalhostSvc.UpdateConfig(*svcConfig)
								if err != nil {
									t.showErr(err, nil)
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
							devStartOps.LocalSyncDir = []string{str}
							devStartOps.NoTerminal = true

							writer := t.switchBodyToScrollingView(" Start DevMode", workloadListTable)
							RedirectionOut(writer)
							go func() {
								if ops == startDupDevModeOpt {
									devStartOps.DevModeType = "duplicate"
								}
								d := dev.DevStartOps{DevStartOptions: devStartOps}
								d.StartDevMode(appMeta.Application)
								RecoverOut()

								podList, err := nocalhostSvc.GetPodList()
								if err != nil {
									t.showErr(err, func() {})
									return
								}
								var runningPod = make([]v1.Pod, 0, 1)
								for _, item := range podList {
									if item.Status.Phase == v1.PodRunning && item.DeletionTimestamp == nil {
										runningPod = append(runningPod, item)
									}
								}
								if len(runningPod) != 1 {
									t.showErr(
										errors.New(fmt.Sprintf("Pod num is %d (not 1), please specify one", len(runningPod))), nil)
									return
								}
								t.app.Suspend(func() {
									fmt.Print("\033[H\033[2J") // clear screen
									nocalhostSvc.EnterPodTerminal(runningPod[0].Name, "nocalhost-dev", "bash", _const.DevModeTerminalBanner)
									t.QueueUpdateDraw(func() {
										t.app.SetFocus(t.body.ItemAt(1))
									})
								})

							}()
						}()

					})
				case endDevModeOpt:
					writer := t.switchBodyToScrollingView(" End DevMode", workloadListTable)
					RedirectionOut(writer)
					go func() {
						dev.EndDevMode(nocalhostSvc)
						RecoverOut()
					}()
				case portForwardOpt:
					inputField := tview.NewInputField()
					inputField.SetBorderPadding(0, 0, 0, 0)
					inputField.SetBorder(true)
					inputField.SetBorderFocusColor(focusBorderColor)
					inputField.SetBackgroundColor(backgroundColor)
					inputField.SetFieldBackgroundColor(backgroundColor)
					inputField.SetPlaceholderTextColor(tcell.ColorLime)
					inputField.SetPlaceholder("Input a PortForward(eg: 1234:1234)")

					//inputField
					f := tview.NewFlex()
					f.SetBackgroundColor(backgroundColor)
					f.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						switch event.Key() {
						case tcell.KeyEscape:
							t.pages.HidePage("pfPage")
							t.app.SetFocus(workloadListTable)
						case tcell.KeyTab:
							if f.ItemAt(1).HasFocus() {
								t.app.SetFocus(f.ItemAt(0))
							} else {
								t.app.SetFocus(f.ItemAt(1))
							}
						}
						return event
					})
					f.SetDirection(tview.FlexRow)
					f.AddItem(inputField, 3, 1, true)
					_, _, w, h := t.mainLayout.GetRect()
					f.SetRect(w/2-25, h/2-10, 50, 20)
					pfListTable := NewRowSelectableTable("")

					pfListTableHeaders := []string{"Port", "Status", "Pod"}
					for i, header := range pfListTableHeaders {
						pfListTable.SetCellSimple(0, i, header)
						pfListTable.GetCell(0, i).SetSelectable(false).SetMaxWidth(12).SetAlign(tview.AlignCenter)
					}

					updatePfListFunc := func() {
						// find port forward
						pro, err := nocalhostSvc.GetProfile()
						if err != nil {
							t.showErr(err, nil)
							return
						}
						pfListTable.Clear()
						for i, header := range pfListTableHeaders {
							pfListTable.SetCellSimple(0, i, header)
							pfListTable.GetCell(0, i).SetSelectable(false).SetMaxWidth(12).SetAlign(tview.AlignCenter)
						}
						for i2, forward := range pro.DevPortForwardList {
							pfListTable.SetCell(i2+1, 0, coloredCell(fmt.Sprintf("%d:%d", forward.LocalPort, forward.RemotePort)))
							pfListTable.SetCell(i2+1, 1, coloredCell(forward.Status))
							pfListTable.SetCell(i2+1, 2, coloredCell(forward.PodName))
						}
					}

					updatePfListFunc()
					pfListTable.SetSelectedFunc(func(row, column int) {
						pf := pfListTable.GetCell(row, 0).Text
						t.ConfirmationBox(fmt.Sprintf("Do you want to stop PortForward %s?", pf), func() {
							err := nocalhostSvc.StopPortForwardByPort(pf)
							if err != nil {
								t.showErr(err, nil)
							} else {
								t.InfoBox(fmt.Sprintf("%s port-forward has been stop", pf), func() {
									time.Sleep(1 * time.Second)
									go func() {
										t.app.QueueUpdateDraw(updatePfListFunc)
									}()
								})
							}
						}, func() {
							t.app.SetFocus(pfListTable)
						})
					})

					inputField.SetChangedFunc(func(text string) {
						reg1 := regexp.MustCompile("[^0-9:]")
						if reg1.MatchString(text) {
							inputField.SetText(reg1.ReplaceAllLiteralString(text, ""))
						}
					})
					inputField.SetDoneFunc(func(key tcell.Key) {
						if key == tcell.KeyEnter {
							pfStr := inputField.GetText()
							l, r, err := utils.GetPortForwardForString(pfStr)
							if err != nil {
								t.showErr(err, nil)
								return
							}

							// Select a pods
							pl, err := getPodNameList()
							if err != nil {
								t.showErr(err, nil)
								return
							}
							t.buildReusableTable(pl, func(podName string) {
								err = nocalhostSvc.PortForward(podName, l, r, "")
								if err != nil {
									t.showErr(err, nil)
									return
								}
								t.InfoBox(fmt.Sprintf("%s port-forward has been started", pfStr), func() {
									inputField.SetText("")
									go func() {
										time.Sleep(1 * time.Second)
										t.app.QueueUpdateDraw(updatePfListFunc)
									}()
								})
							})
						}
					})

					f.AddItem(pfListTable, 0, 1, true)
					f.SetBorder(true)
					t.pages.AddPage("pfPage", f, false, true)
					t.pages.ShowPage("pfPage")
					t.app.SetFocus(f)
				case viewDevConfigOpt:
					writer := t.switchBodyToScrollingView("Dev Config", workloadListTable)
					go func() {
						config := nocalhostSvc.Config()
						bys, err := yaml.Marshal(config)
						if err != nil {
							t.showErr(err, nil)
							return
						}
						writer.Write(bys)
					}()
				case openTerminalOpt:
					go func() {
						podList, err := nocalhostSvc.GetPodList()
						if err != nil {
							t.showErr(err, nil)
							return
						}
						var runningPod = make([]v1.Pod, 0, 1)
						for _, item := range podList {
							if item.Status.Phase == v1.PodRunning && item.DeletionTimestamp == nil {
								runningPod = append(runningPod, item)
							}
						}
						if len(runningPod) == 0 {
							t.showErr(errors.New(fmt.Sprintf("Pod num is 0 ?")), nil)
							return
						}

						if nocalhostSvc.IsInDevMode() {
							t.app.Suspend(func() {
								fmt.Print("\033[H\033[2J") // clear screen
								nocalhostSvc.EnterPodTerminal(runningPod[0].Name, "nocalhost-dev", "bash", _const.DevModeTerminalBanner)
							})
							t.app.QueueUpdateDraw(func() {
								t.app.SetFocus(workloadListTable)
							})
						} else {
							names := make([]string, 0)
							for _, pod := range runningPod {
								names = append(names, pod.Name)
							}
							t.app.SetFocus(workloadListTable)
							t.buildReusableTable(names, func(selectedPod string) {
								var sp *v1.Pod
								for _, pod := range runningPod {
									if pod.Name == selectedPod {
										sp = &pod
									}
								}
								if sp == nil {
									t.ShowInfo(fmt.Sprintf("Pod %s is not exist?", selectedPod))
									return
								}
								cs := make([]string, 0)
								for _, container := range sp.Spec.Containers {
									cs = append(cs, container.Name)
								}
								t.buildReusableTable(cs, func(selectedItem string) {
									t.app.Suspend(func() {
										fmt.Print("\033[H\033[2J") // clear screen
										nocalhostSvc.EnterPodTerminal(selectedPod, selectedItem, "bash", _const.DevModeTerminalBanner)
									})
									t.app.QueueUpdateDraw(func() {
										t.app.SetFocus(workloadListTable)
									})
								})
							})
						}
					}()
				case viewLogsOpt:
					// get pod list
					podList, err := nocalhostSvc.GetPodList()
					if err != nil {
						t.showErr(err, nil)
						return
					}

					showContainerListFunc := func(podName string) {
						var selectedPod *v1.Pod
						for _, pod := range podList {
							if pod.Name == podName {
								selectedPod = &pod
							}
						}
						containers := make([]string, 0)
						if selectedPod == nil {
							t.ShowInfo("No selected Pod??")
							return
						}

						for _, container := range selectedPod.Spec.Containers {
							containers = append(containers, container.Name)
						}

						containerTable := t.pageWithTable("ContainersPage", "Containers", containers, 50, 10)
						containerTable.SetSelectedFunc(func(row, column int) {
							t.pages.HidePage("ContainersPage")
							t.app.SetFocus(workloadListTable)
							writer := t.switchBodyToScrollingView(" View logs", workloadListTable)
							kube.InitLogOptions()
							kube.LogOptions.IOStreams.Out = writer
							// nhctl k logs --tail=3000 reviews-6446f84d54-8hdrt --namespace nocalhost-aaaa -c reviews --kubeconfig xxx
							kube.LogOptions.Container = containerTable.GetCell(row, column).Text
							kube.LogOptions.Tail = 3000
							go func() {
								// todo: we may need to recreate LogOptions
								kube.RunLogs(kube.CmdLogs, []string{podName})
							}()
						})
					}
					if len(podList) > 1 {
						podNames := make([]string, 0)
						for _, pod := range podList {
							podNames = append(podNames, pod.Name)
						}
						podListTable := t.pageWithTable("PodNameListPage", "Pod", podNames, 50, 10)
						podListTable.SetSelectedFunc(func(row, column int) {
							t.pages.HidePage("PodNameListPage")
							t.app.SetFocus(workloadListTable)
							showContainerListFunc(podListTable.GetCell(row, column).Text)
						})
					} else {
						showContainerListFunc(podList[0].Name)
					}

				case syncLogsOpt:
					logPath := filepath.Join(nocalhostSvc.GetSyncDir(), "syncthing.log")
					if _, err = os.Stat(logPath); err != nil {
						t.showErr(err, nil)
						return
					}
					writer := t.switchBodyToScrollingView(" File Sync logs", workloadListTable)
					go func() {
						bys, err := ioutil.ReadFile(logPath)
						if err != nil {
							t.showErr(err, nil)
							return
						}
						if _, err := writer.Write(bys); err != nil {
							t.showErr(err, nil)
							return
						}
					}()
				case openGuiOpt:
					guiPortCell := workloadListTable.GetCell(row, 5)
					if guiPortCell == nil || guiPortCell.Text == "" || guiPortCell.Text == "0" {
						t.showErr(errors.New("file sync is not running"), nil)
						return
					}
					go func() {
						exec.Command(`open`, fmt.Sprintf("http://localhost:%s", guiPortCell.Text)).Start()
					}()
				case viewProfile:
					pro, err := nocalhostSvc.GetProfile()
					if err != nil {
						t.showErr(err, nil)
						return
					}
					w := t.switchBodyToScrollingView("Profile", workloadListTable)
					bys, _ := yaml.Marshal(pro)
					go func() {
						_, err = w.Write(bys)
						if err != nil {
							t.showErr(err, nil)
							return
						}
					}()
				case viewDBData:
					appName := nocalhostApp.Name
					nid := nocalhostApp.GetAppMeta().NamespaceId
					result, err := nocalhost.ListAllFromApplicationDb(ns, appName, nid)
					if err != nil {
						t.showErr(err, nil)
						return
					}
					w := t.switchBodyToScrollingView("DB Data", workloadListTable)
					go func() {
						for key, val := range result {
							_, err = w.Write([]byte(fmt.Sprintf("%s=%s\n", key, val)))
							if err != nil {
								t.ShowInfo(err.Error())
							}
						}
					}()
				}

			})
		}
	}
	workloadListTable.SetSelectedFunc(selectWorkloadFunc)
	workloadListTable.Select(1, 0)
	return workloadListTable
}

//func (t *TviewApplication) ShowOriginalContainersTable(x, y, width, height int) *EnhancedTable {
//	containerList, err := nocalhostSvc.GetOriginalContainers()
//	if err != nil {
//		t.showErr(err, nil)
//		return nil
//	}
//	return t.containersTable(containerList, x, y, width, height)
//}

func (t *TviewApplication) containersTable(containerList []v1.Container, x, y, width, height int) *EnhancedTable {
	containerTable := t.NewBorderedTable("Containers")
	for i, container := range containerList {
		containerTable.SetCellSimple(i, 0, container.Name)
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
		}
		return event
	})
	return containerTable
}

func (t *TviewApplication) getCenterRect(width, height int) (int, int, int, int) {
	_, _, w, h := t.mainLayout.GetRect()
	return w/2 - width/2, h/2 - height/2, width, height
}

func (t *TviewApplication) pageWithTable(pageName, tableName string, rows []string, width, height int) *EnhancedTable {
	table := t.NewBorderedTable(tableName)
	for i, row := range rows {
		table.SetCellSimple(i, 0, row)
	}
	table.SetRect(t.getCenterRect(width, height))
	t.app.SetFocus(table)
	t.pages.AddPage(pageName, table, false, true)
	t.pages.ShowPage(pageName)
	table.SetBlurFunc(func() {
		t.pages.RemovePage(pageName)
	})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			t.pages.RemovePage(pageName)
			return nil
		}
		return event
	})
	return table
}

func (t *TviewApplication) NewCentralPage(pageName string, p tview.Primitive, width, height int) func() {
	flex := tview.NewFlex()
	flex.AddItem(p, 0, 1, true)
	ep := &EnhancedPrimitive{Primitive: flex}
	t.pages.AddPage(pageName, ep, false, true)
	t.pages.ShowPage(pageName)
	flex.SetRect(t.getCenterRect(width, height))
	ep.SetBlurFunc(func() {
		t.pages.RemovePage(pageName)
	})
	flex.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc {
			t.pages.RemovePage(pageName)
			return nil
		}
		return event
	})
	t.app.SetFocus(p)
	return func() {
		t.pages.RemovePage(pageName)
	}
}

func (t *TviewApplication) getRect() (int, int, int, int) {
	return t.mainLayout.GetRect()
}

func (t *TviewApplication) buildReusableTable(items []string, f func(selectedItem string)) {
	pageName := "ReusablePage"
	tab := NewRowSelectableTable("<Pod>")
	tab.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			t.pages.RemovePage(pageName)
		}
		return event
	})
	for i, it := range items {
		tab.SetCell(i, 0, coloredCell(it))
	}
	_, _, w, h := t.getRect()
	tab.SetRect(w/2-25, h/2-5, 50, 10)
	tab.SetSelectedFunc(func(row, column int) {
		t.pages.RemovePage(pageName)
		if f != nil {
			tx := tab.GetCell(row, column).Text
			f(tx)
		}
	})
	t.app.SetFocus(tab)
	if t.pages.HasPage(pageName) {
		t.pages.RemovePage(pageName)
	}
	t.QueueUpdateDraw(func() {
		t.pages.AddPage(pageName, tab, false, true)
		t.pages.ShowPage(pageName)
	})
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
