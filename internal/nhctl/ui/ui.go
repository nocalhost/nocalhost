/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"context"
	"fmt"
	"github.com/gdamore/tcell/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"

	//"github.com/rivo/tview"
	"github.com/derailed/tview"
)

const (
	clusterInfoWidth = 50
	clusterInfoPad   = 15

	// Main Menu
	deployApplicationOption = " Deploy Application"
	switchContext           = " Switch Context"

	deployDemoAppOption      = " Quickstart: Deploy BookInfo demo application"
	deployHelmAppOption      = " Helm: Use my own Helm chart (e.g. local via ./chart/ or any remote chart)"
	deployKubectlAppOption   = " Kubectl: Use existing Kubernetes manifests (e.g. ./kube/deployment.yaml)"
	deployKustomizeAppOption = " Kustomize: Use an existing Kustomization (e.g. ./kube/kustomization/)"
)

func RunTviewApplication() {
	app := NewTviewApplication()
	if err := app.Run(); err != nil {
		return
	}
}

func (t *TviewApplication) buildMainMenu() tview.Primitive {
	mainMenu := NewBorderTable(" Menu")
	mainMenu.SetCell(0, 0, coloredCell(deployApplicationOption))
	mainMenu.SetCell(1, 0, coloredCell(" Use a existing application"))
	mainMenu.SetCell(2, 0, coloredCell(" Start DevMode Here"))
	mainMenu.SetCell(3, 0, coloredCell(switchContext))
	mainMenu.SetCell(4, 0, coloredCell(" view"))

	// Make selected eventHandler the same as clicked
	mainMenu.SetSelectedFunc(func(row, column int) {
		selectedCell := mainMenu.GetCell(row, column)
		var m tview.Primitive
		switch selectedCell.Text {
		case deployApplicationOption:
			m = t.buildDeployApplicationMenu()
		case switchContext:
			m = t.buildSelectContextMenu()
		default:
			return
		}
		t.switchBodyTo(m)
	})
	return mainMenu
}

func (t *TviewApplication) buildSelectContextMenu() *tview.Table {
	table := NewBorderTable(" Select a context")

	for col, section := range []string{" Context", "Cluster", "User", "NameSpace", "K8s Rev"} {
		table.SetCell(0, col, infoCell(section))
	}
	cs := loadAllClusterInfos()
	for i, c := range cs {
		table.SetCell(i+1, 0, coloredCell(" "+c.Context))
		table.SetCell(i+1, 1, coloredCell(c.Cluster))
		table.SetCell(i+1, 2, coloredCell(c.User))
		table.SetCell(i+1, 3, coloredCell(c.NameSpace))
		table.SetCell(i+1, 4, coloredCell(c.K8sVer))
	}

	table.SetSelectedFunc(func(row, column int) {
		if row > 0 {
			if len(cs) >= row {
				t.clusterInfo = cs[row-1]
				if !t.clusterInfo.k8sClient.IsClusterAdmin() {
					t.RefreshHeader()
					t.switchMainMenu()
					return
				}
				ns, err := t.clusterInfo.k8sClient.ClientSet.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
				if err != nil {
					return
				}
				nsTable := NewBorderTable("Select a namespace")
				for i, item := range ns.Items {
					nsTable.SetCell(i, 0, coloredCell(" "+item.Name))
				}
				nsTable.SetSelectedFunc(func(row, column int) {
					cell := nsTable.GetCell(row, column)
					t.clusterInfo.NameSpace = cell.Text
					t.clusterInfo.k8sClient.NameSpace(cell.Text)
					t.RefreshHeader()
					t.switchMainMenu()
				})
				t.switchBodyTo(nsTable)
			}
		}
	})

	table.Select(1, 0)
	return table
}

func (t *TviewApplication) buildView() tview.Primitive {
	view := tview.NewTextView()
	view.SetTitle(" Text view")
	view.SetBorder(true)
	view.SetText(" hhh")
	view.SetScrollable(true)
	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			go func() {
				for i := 0; i < 100; i++ {
					time.Sleep(1 * time.Second)
					ttx := view.GetText(true)
					t.app.QueueUpdateDraw(func() {
						view.SetText(fmt.Sprintf("%d %s\n", i, ttx))
					})
					t.app.SetFocus(view)
				}
			}()
		}
		return event
	})
	return view
}

func (t *TviewApplication) buildDeployApplicationMenu() *tview.Table {
	menu := NewBorderTable(" Deploy Application")
	menu.SetCell(0, 0, coloredCell(deployDemoAppOption))
	menu.SetCell(1, 0, coloredCell(deployHelmAppOption))
	menu.SetCell(2, 0, coloredCell(deployKubectlAppOption))
	menu.SetCell(3, 0, coloredCell(deployKustomizeAppOption))

	menu.SetSelectedFunc(func(row, column int) {
		selectedCell := menu.GetCell(row, column)
		var m tview.Primitive
		switch selectedCell.Text {
		case deployDemoAppOption:
			view := tview.NewTextView()
			view.SetTitle(" Deploy BookInfo demo application")
			view.SetBorder(true)
			view.SetText(fmt.Sprintf(" nhctl install bookinfo --git-url https://github.com/nocalhost/bookinfo.git --type rawManifest --kubeconfig %s --namespace %s\n", t.clusterInfo.KubeConfig, t.clusterInfo.NameSpace))
			view.SetScrollable(true)
			m = view
		default:
			return
		}
		t.switchBodyTo(m)
	})
	return menu
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
