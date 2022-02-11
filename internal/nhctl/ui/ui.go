/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"github.com/gdamore/tcell/v2"
	"strings"
	"time"

	//"github.com/rivo/tview"
	"github.com/derailed/tview"
)

const (
	clusterInfoWidth = 50
	clusterInfoPad   = 15

	deployDemoAppOption      = "Quickstart: Deploy BookInfo demo application"
	deployHelmAppOption      = "Helm: Use my own Helm chart (e.g. local via ./chart/ or any remote chart)"
	deployKubectlAppOption   = "Kubectl: Use existing Kubernetes manifests (e.g. ./kube/deployment.yaml)"
	deployKustomizeAppOption = "Kustomize: Use an existing Kustomization (e.g. ./kube/kustomization/)"

	startDevModeOpt    = "Start DevMode"
	startDupDevModeOpt = "Start DevMode(Duplicate)"
	endDevModeOpt      = "End DevMode"
	viewDevConfigOpt   = "View Dev Config"
	viewLogsOpt        = "View Logs"
	portForwardOpt     = "Port Forward"
	syncLogsOpt        = "File Sync Logs"
	openGuiOpt         = "Open Sync GUI"
	openTerminalOpt    = "Open Terminal"
	viewProfile        = "View Profile"
	viewDBData         = "View DB Data"
)

func RunTviewApplication() {
	app := NewTviewApplication()
	if app == nil {
		return
	}
	stopChan := make(chan struct{})
	go func() {
		if err := app.Run(); err != nil {
			stopChan <- struct{}{}
		}
	}()

	go func() {
		lp := getLastPosition()
		if lp != "" {
			strs := strings.Split(lp, "/")
			if len(strs) == 0 {
				return
			}

			if app.treeInBody == nil {
				return
			}
			app.app.SetFocus(app.treeInBody)
			r := app.treeInBody.GetRoot()
			if r == nil {
				return
			}

			for _, str := range strs {
				var found bool
				for _, node := range r.GetChildren() {
					if GetText(node) == str {
						app.treeInBody.SetCurrentNode(node)
						r = node
						go func() {
							app.app.QueueEvent(tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone))
						}()
						for i := 0; i < 200; i++ {
							time.Sleep(100 * time.Millisecond)
							if len(r.GetChildren()) > 0 {
								ExpandText(node)
								node.SetExpanded(true)
								break
							}
						}
						found = true
						break
					}
				}
				if !found {
					return
				}
			}
		}
	}()

	<-stopChan
}

func (t *TviewApplication) buildSelectContextMenu() *EnhancedTable {
	pageName := "ClusterPage"
	table := NewRowSelectableTable("Select a context")

	for col, section := range []string{"Context", "Cluster", "User", "NameSpace", "K8s Rev"} {
		table.SetCell(0, col, infoCell(section))
	}
	cs := loadAllClusterInfos()
	for i, c := range cs {
		table.SetCell(i+1, 0, coloredCell(c.Context))
		table.SetCell(i+1, 1, coloredCell(c.Cluster))
		table.SetCell(i+1, 2, coloredCell(c.User))
		table.SetCell(i+1, 3, coloredCell(c.NameSpace))
		table.SetCell(i+1, 4, coloredCell(c.K8sVer))
	}

	table.SetSelectedFunc(func(row, column int) {
		if row > 0 {
			if len(cs) >= row {
				t.clusterInfo = cs[row-1]
				//if !t.clusterInfo.k8sClient.IsClusterAdmin() {
				t.RefreshHeader()
				t.QueueUpdateDraw(func() {
					t.buildTreeBody()
				})
				t.pages.RemovePage(pageName)
				return
				//}
			}
		}
	})
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			t.pages.RemovePage(pageName)
			return nil
		}
		return event
	})
	table.SetRect(t.getCenterRect(100, 10))
	t.app.SetFocus(table)
	t.pages.AddPage(pageName, table, false, true)
	table.Select(1, 0)
	return table
}

type SyncBuilder struct {
	WriteFunc func(p []byte) (int, error)
}

func (s *SyncBuilder) Write(p []byte) (int, error) {
	return s.WriteFunc(p)
}

func (s *SyncBuilder) Sync() error {
	return nil
}

func coloredCell(t string) *tview.TableCell {
	cell := tview.NewTableCell(t)
	cell.SetTextColor(tcell.ColorPaleGreen)
	return cell
}

func infoCell(t string) *tview.TableCell {
	cell := tview.NewTableCell(t)
	cell.SetExpansion(2)
	return cell
}

func sectionCell(t string) *tview.TableCell {
	cell := tview.NewTableCell(t + ":")
	cell.SetAlign(tview.AlignLeft)
	cell.SetTextColor(tcell.ColorPaleGreen)
	return cell
}

func keyCell(t string) *tview.TableCell {
	cell := tview.NewTableCell("<" + t + ">")
	cell.SetAlign(tview.AlignLeft)
	cell.SetTextColor(tcell.ColorPaleGreen)
	return cell
}
