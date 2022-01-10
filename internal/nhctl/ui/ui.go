/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"context"
	"github.com/gdamore/tcell/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/nocalhost"
	"strings"
	//"github.com/rivo/tview"
	"github.com/derailed/tview"
)

const (
	clusterInfoWidth = 50
	clusterInfoPad   = 15

	// Main Menu
	deployApplicationOption = " Deploy Application"
	switchContextOption     = " Switch Context"
	selectAppOption         = " List applications"

	deployDemoAppOption      = " Quickstart: Deploy BookInfo demo application"
	deployHelmAppOption      = " Helm: Use my own Helm chart (e.g. local via ./chart/ or any remote chart)"
	deployKubectlAppOption   = " Kubectl: Use existing Kubernetes manifests (e.g. ./kube/deployment.yaml)"
	deployKustomizeAppOption = " Kustomize: Use an existing Kustomization (e.g. ./kube/kustomization/)"

	startDevModeOpt = " Start DevMode"
	viewLogsOpt     = " View Logs"
)

func RunTviewApplication() {
	app := NewTviewApplication()
	if err := app.Run(); err != nil {
		return
	}
}

func (t *TviewApplication) buildMainMenu() tview.Primitive {
	mainMenu := NewBorderedTable(" Menu")
	mainMenu.SetCell(0, 0, coloredCell(deployApplicationOption))
	mainMenu.SetCell(1, 0, coloredCell(selectAppOption))
	mainMenu.SetCell(2, 0, coloredCell(" Start DevMode Here"))
	mainMenu.SetCell(3, 0, coloredCell(switchContextOption))

	// Make selected eventHandler the same as clicked
	mainMenu.SetSelectedFunc(func(row, column int) {
		selectedCell := mainMenu.GetCell(row, column)
		var m tview.Primitive
		switch selectedCell.Text {
		case deployApplicationOption:
			m = t.buildDeployApplicationMenu()
		case switchContextOption:
			m = t.buildSelectContextMenu()
		case selectAppOption:
			m = t.buildSelectAppList()
		default:
			return
		}
		t.switchBodyToC(mainMenu, m)
	})
	return mainMenu
}

func (t *TviewApplication) buildSelectAppList() *tview.Table {
	selectAppTable := NewBorderedTable(" Select a application")

	metas, err := nocalhost.GetApplicationMetas(t.clusterInfo.NameSpace, t.clusterInfo.KubeConfig)
	if err != nil {
		panic(err)
	}

	for col, section := range []string{" Application", "State", "Type"} {
		selectAppTable.SetCell(0, col, infoCell(section))
	}

	for i, c := range metas {
		selectAppTable.SetCell(i+1, 0, coloredCell(" "+c.Application))
		selectAppTable.SetCell(i+1, 1, coloredCell(string(c.ApplicationState)))
		selectAppTable.SetCell(i+1, 2, coloredCell(string(c.ApplicationType)))
	}

	selectAppTable.SetSelectedFunc(func(row, column int) {
		if row > 0 {
			nextTable := t.buildSelectWorkloadList(metas[row-1])
			t.switchBodyToC(selectAppTable, nextTable)
		}
	})
	selectAppTable.Select(1, 0)
	return selectAppTable
}

func showErr(err error) {
	panic(err)
}

func (t *TviewApplication) buildSelectContextMenu() *tview.Table {
	table := NewBorderedTable(" Select a context")

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
				nsTable := NewBorderedTable("Select a namespace")
				for i, item := range ns.Items {
					nsTable.SetCell(i, 0, coloredCell(" "+item.Name))
				}
				nsTable.SetSelectedFunc(func(row, column int) {
					cell := nsTable.GetCell(row, column)
					t.clusterInfo.NameSpace = strings.Trim(cell.Text, " ")
					t.clusterInfo.k8sClient.NameSpace(strings.Trim(cell.Text, " "))
					t.RefreshHeader()
					t.switchMainMenu()
				})
				t.switchBodyToC(table, nsTable)
			}
		}
	})

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
