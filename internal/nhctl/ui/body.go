/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"context"
	"github.com/derailed/tview"
	"github.com/gdamore/tcell/v2"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"nocalhost/internal/nhctl/nocalhost"
	yaml "nocalhost/pkg/nhctl/utils/custom_yaml_v3"
	"os"
	"strings"
)

var (
	rightBodyProportion = 3
	leftBodyProportion  = 1
	lastPosition        = ""
)

func updateLastPosition(l string) {
	_, err := os.Stat(cliProfileName)
	if err != nil && os.IsNotExist(err) {
		ioutil.WriteFile(cliProfileName, []byte(""), 0644)
	}
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

	tree := tview.NewTreeView()
	tree.SetBorder(true)
	root := tview.NewTreeNode(t.clusterInfo.Cluster).SetSelectable(true)
	tree.SetRoot(root)
	tree.SetCurrentNode(root)

	nsList, err := t.clusterInfo.k8sClient.ClientSet.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.showErr(err.Error())
		return
	}

	for _, item := range nsList.Items {
		nsNode := tview.NewTreeNode(item.Name).SetSelectable(true)
		nsNode.SetSelectedFunc(func() {
			lastPosition = nsNode.GetText()
			if nsNode.IsExpanded() && len(nsNode.GetChildren()) > 0 {
				nsNode.Collapse()
				return
			}
			metas, err := nocalhost.GetApplicationMetas(nsNode.GetText(), t.clusterInfo.KubeConfig)
			if err != nil {
				t.showErr(err.Error())
				return
			}

			nsNode.ClearChildren()
			for _, meta := range metas {
				appNode := tview.NewTreeNode(meta.Application)
				gs := []string{"Workloads", "CustomResources", "Network", "Configuration", "Storage"}
				for _, g := range gs {
					groupNode := tview.NewTreeNode(g)
					if g == "Workloads" {
						wls := []string{"Deployments", "DaemonSets", "StatefulSets", "Jobs", "CronJobs", "Pods"}
						for _, wl := range wls {
							wlNode := tview.NewTreeNode(wl)
							m := meta
							wlNode.SetSelectedFunc(func() {
								lastPosition = strings.Join([]string{nsNode.GetText(), appNode.GetText(), groupNode.GetText(), wlNode.GetText()}, "/")
								table := t.buildSelectWorkloadList(m, nsNode.GetText(), wlNode.GetText())
								flex.RemoveItemAtIndex(1)
								flex.AddItem(table, 0, 3, true)
								t.app.SetFocus(table)
							})
							groupNode.AddChild(wlNode)
						}
					}
					groupNode.SetExpanded(false)
					groupNode.SetSelectedFunc(func() {
						lastPosition = strings.Join([]string{nsNode.GetText(), appNode.GetText(), groupNode.GetText()}, "/")
						groupNode.SetExpanded(!groupNode.IsExpanded())
					})
					appNode.AddChild(groupNode)
				}
				appNode.SetExpanded(false)
				appNode.SetSelectedFunc(func() {
					lastPosition = strings.Join([]string{nsNode.GetText(), appNode.GetText()}, "/")
					appNode.SetExpanded(!appNode.IsExpanded())
				})
				nsNode.AddChild(appNode)
			}
			nsNode.SetExpanded(true)
		})
		root.AddChild(nsNode)
	}
	tree.SetBorderFocusColor(tcell.ColorPaleGreen)

	table := NewBorderedTable("")
	flex.AddItem(tree, 0, leftBodyProportion, true)
	flex.AddItem(table, 0, rightBodyProportion, false)
	flex.SetBorder(true)

	t.app.SetFocus(tree)
	t.treeInBody = tree
	t.rightInBody = table
	t.body = flex
	t.body.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			t.app.SetFocus(t.treeInBody)
			return nil
		case tcell.KeyRight:
			item := t.body.ItemAt(1)
			if item != nil {
				t.app.SetFocus(item)
			}
			return nil
		}
		return event
	})
}
