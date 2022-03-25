/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"context"
	"fmt"
	"github.com/derailed/tview"
	"github.com/gdamore/tcell/v2"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/cmd/nhctl/cmds/install"
	"nocalhost/internal/nhctl/app_flags"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/common"
	"nocalhost/internal/nhctl/nocalhost"
	"nocalhost/pkg/nhctl/log"
	yaml "nocalhost/pkg/nhctl/utils/custom_yaml_v3"
	"os"
	"strings"
)

var (
	rightBodyProportion = 3
	leftBodyProportion  = 1
	lastPosition        = ""

	expandedPrefix = "v "
	collapsePrefix = "> "
)

func createCliProfileIfNotExist() {
	_, err := os.Stat(cliProfileDir)
	if err != nil && os.IsNotExist(err) {
		os.MkdirAll(cliProfileDir, 0755)
	}

	_, err = os.Stat(cliProfileName)
	if err != nil && os.IsNotExist(err) {
		ioutil.WriteFile(cliProfileName, []byte(""), 0644)
	}
}

func updateLastPosition(l string) {
	createCliProfileIfNotExist()
	bys, err := ioutil.ReadFile(cliProfileName)
	if err != nil {
		return
	}
	profile := &CliProfile{}
	if err := yaml.Unmarshal(bys, profile); err != nil {
		return
	}
	profile.LastPosition = l
	bys, _ = yaml.Marshal(profile)
	ioutil.WriteFile(cliProfileName, bys, 0644)
}

func getLastPosition() string {
	createCliProfileIfNotExist()
	bys, err := ioutil.ReadFile(cliProfileName)
	if err != nil {
		return ""
	}
	profile := &CliProfile{}
	if err := yaml.Unmarshal(bys, profile); err != nil {
		return ""
	}
	return profile.LastPosition
}

func (t *TviewApplication) buildTreeBody() {
	flex := tview.NewFlex()
	flex.SetDirection(tview.FlexColumn)
	flex.SetBackgroundColor(backgroundColor)

	tree := tview.NewTreeView()
	tree.SetBorder(true)
	root := NewTreeNode(t.clusterInfo.Cluster)
	tree.SetRoot(root)
	tree.SetCurrentNode(root)
	tree.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {

		if event.Key() == tcell.KeyCtrlD {
			cn := tree.GetCurrentNode()
			if cn == nil {
				return nil
			}
			if cn.GetReference() == nil {
				return nil
			}
			s, ok := cn.GetReference().(string)
			if !ok || s != "namespace" {
				return nil
			}

			menu := t.NewBorderedTable("Deploy Application")
			menu.SetCell(0, 0, coloredCell(deployDemoAppOption))
			menu.SetCell(1, 0, coloredCell(deployHelmAppOption))
			menu.SetCell(2, 0, coloredCell(deployKubectlAppOption))
			menu.SetCell(3, 0, coloredCell(deployKustomizeAppOption))
			removeFunc := t.NewCentralPage("DeployPage", menu, 100, 10)
			menu.SetSelectedFunc(func(row, column int) {
				selectedCell := menu.GetCell(row, column)
				switch selectedCell.Text {
				case deployDemoAppOption:
					removeFunc()
					sbd := t.switchBodyToScrollingView("", nil)
					//nhctl install bookinfo --git-url https://github.com/nocalhost/bookinfo.git --type rawManifest --kubeconfig %s --namespace %s
					f := app_flags.InstallFlags{
						GitUrl:  "https://github.com/nocalhost/bookinfo.git",
						AppType: string(appmeta.ManifestGit),
					}
					log.RedirectionDefaultLogger(sbd)
					go func() {
						_, err := common.InstallApplication(&f, "bookinfo", t.clusterInfo.KubeConfig, GetText(cn))
						if err != nil {
							t.ShowInfo(err.Error())
						}
						log.RedirectionDefaultLogger(os.Stdout)
					}()
				default:
					return
				}
			})
			return nil
		} else if event.Key() == tcell.KeyCtrlU {
			cn := tree.GetCurrentNode()
			if cn == nil {
				return nil
			}
			if cn.GetReference() == nil {
				return nil
			}
			s, ok := cn.GetReference().([]string)
			if !ok || s[0] != "application" {
				return nil
			}

			t.ConfirmationBox(fmt.Sprintf("Do you want to uninstall application %s in %s ?", GetText(cn), s[1]), func() {
				sbd := t.switchBodyToScrollingView("", nil)
				log.RedirectionDefaultLogger(sbd)
				go func() {
					err := install.Uninstall(t.clusterInfo.KubeConfig, s[1], GetText(cn))
					if err != nil {
						t.showErr(err, nil)
					}
					log.RedirectionDefaultLogger(os.Stdout)
				}()
			}, nil)
		}
		return event
	})

	nsList, err := t.clusterInfo.k8sClient.ClientSet.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.ShowInfo(err.Error())
		return
	}

	for _, item := range nsList.Items {
		nsNode := NewTreeNode(collapsePrefix + item.Name)
		nsNode.SetReference("namespace")
		nsNode.SetSelectedFunc(func() {
			lastPosition = GetText(nsNode)
			if nsNode.IsExpanded() && len(nsNode.GetChildren()) > 0 {
				CollapseText(nsNode)
				nsNode.Collapse()
				return
			}
			ExpandText(nsNode)
			metas, err := nocalhost.GetApplicationMetas(GetText(nsNode), t.clusterInfo.KubeConfig)
			if err != nil {
				t.showErr(err, nil)
				return
			}

			nsNode.ClearChildren()
			for _, meta := range metas {
				appNode := NewTreeNode(meta.Application)
				appNode.SetReference([]string{"application", GetText(nsNode)})
				CollapseText(appNode)
				gs := []string{"Workloads", "CustomResources", "Network", "Configuration", "Storage"}
				for _, g := range gs {
					groupNode := NewTreeNode(g)
					CollapseText(groupNode)
					if g == "Workloads" {
						wls := []string{"Deployments", "DaemonSets", "StatefulSets", "Jobs", "CronJobs", "Pods"}
						for _, wl := range wls {
							wlNode := NewTreeNode(wl)
							m := meta
							wlNode.SetSelectedFunc(func() {
								lastPosition = strings.Join([]string{GetText(nsNode), GetText(appNode), GetText(groupNode), GetText(wlNode)}, "/")

								go func() {
									table := t.buildWorkloadList(m, GetText(nsNode), GetText(wlNode))
									flex.RemoveItemAtIndex(1)
									flex.AddItem(table, 0, 3, true)
									t.app.QueueUpdateDraw(func() {
										t.app.SetFocus(table)
									})
								}()
							})
							groupNode.AddChild(wlNode)
						}
					}
					groupNode.SetExpanded(false)
					groupNode.SetSelectedFunc(func() {
						lastPosition = strings.Join([]string{GetText(nsNode), GetText(appNode), GetText(groupNode)}, "/")
						if groupNode.IsExpanded() {
							CollapseText(groupNode)
						} else {
							ExpandText(groupNode)
						}
						groupNode.SetExpanded(!groupNode.IsExpanded())
					})
					appNode.AddChild(groupNode)
				}
				appNode.SetExpanded(false)
				appNode.SetSelectedFunc(func() {
					lastPosition = strings.Join([]string{GetText(nsNode), GetText(appNode)}, "/")
					if appNode.IsExpanded() {
						CollapseText(appNode)
					} else {
						ExpandText(appNode)
					}
					appNode.SetExpanded(!appNode.IsExpanded())
				})
				nsNode.AddChild(appNode)
			}
			nsNode.SetExpanded(true)
		})
		root.AddChild(nsNode)
	}
	tree.SetBorderFocusColor(focusBorderColor)

	table := NewRowSelectableTable("")
	flex.AddItem(tree, 0, leftBodyProportion, true)
	flex.AddItem(table, 0, rightBodyProportion, false)

	t.app.SetFocus(tree)
	t.treeInBody = tree
	t.rightInBody = table
	t.body = flex
	t.body.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTab:
			l := LenOfFlex(t.body)
			for i := 0; i < l; i++ {
				it := t.body.ItemAt(i)
				if it.HasFocus() {
					t.ShowInfo("Changing focus...")
					t.app.SetFocus(t.body.ItemAt((i + 1) % l))
					break
				}
			}
			return nil
		}
		return event
	})
	it := t.mainLayout.ItemAt(1)
	if it == nil {
		t.mainLayout.AddItem(t.body, 0, 2, true)
	} else {
		t.mainLayout.RemoveItemAtIndex(1)
		t.mainLayout.AddItemAtIndex(1, t.body, 0, 2, true)
	}
	tree.SetBackgroundColor(backgroundColor)
}

func ExpandText(node *tview.TreeNode) {
	node.SetText(expandedPrefix + GetText(node))
}

func CollapseText(node *tview.TreeNode) {
	node.SetText(collapsePrefix + GetText(node))
}

func GetText(node *tview.TreeNode) string {
	return strings.TrimPrefix(strings.TrimPrefix(node.GetText(), collapsePrefix), expandedPrefix)
}

func (t *TviewApplication) UpdateTreeColor(color tcell.Color) {
	if t.treeInBody == nil {
		return
	}
	root := t.treeInBody.GetRoot()
	root.Walk(func(node, parent *tview.TreeNode) bool {
		node.SetColor(color)
		if parent != nil {
			parent.SetColor(color)
		}
		return true
	})
}
