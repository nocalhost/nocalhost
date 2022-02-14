/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"github.com/derailed/tview"
	"nocalhost/internal/nhctl/daemon_common"
)

func (t *TviewApplication) RefreshHeader() {
	if t.clusterInfo == nil {
		return
	}
	clusterInfo := t.header.ItemAt(0).(*tview.Table)
	clusterInfo.GetCell(0, 1).SetText(t.clusterInfo.Cluster)
	clusterInfo.GetCell(1, 1).SetText(t.clusterInfo.Context)
	clusterInfo.GetCell(2, 1).SetText(t.clusterInfo.User)
	clusterInfo.GetCell(3, 1).SetText(t.clusterInfo.K8sVer)
	clusterInfo.GetCell(4, 1).SetText(daemon_common.Version)

	clWidth := 50
	size := len(t.clusterInfo.Cluster) + clusterInfoPad
	if size > clWidth {
		clWidth = size
	}
	t.header.ResizeItem(clusterInfo, clWidth, 1)
}

func buildHeader() *tview.Flex {
	header := tview.NewFlex()
	header.SetDirection(tview.FlexColumn)

	clWidth := clusterInfoWidth
	clusterName := "minikube"
	size := len(clusterName) + clusterInfoPad
	if size > clWidth {
		clWidth = size
	}

	header.SetBackgroundColor(backgroundColor)
	header.AddItem(clusterInfo(), clWidth, 1, false)
	header.AddItem(keyInfo(), clWidth, 1, false)
	return header
}

func clusterInfo() tview.Primitive {
	table := tview.NewTable()
	table.SetBorderPadding(0, 0, 1, 0)
	for row, section := range []string{"Context", "Cluster", "User", "K8s Rev", "Nhctl Rev"} {
		table.SetCell(row, 0, sectionCell(section))
		table.SetCell(row, 1, infoCell("nil"))
	}
	table.SetBackgroundColor(backgroundColor)
	return table
}

func keyInfo() tview.Primitive {
	table := tview.NewTable()
	table.SetBorderPadding(0, 0, 1, 0)
	keyList := []string{"esc", "tab", "ctrl-c", "ctrl-d", "ctrl-u"}
	descList := []string{"Back or Cancel", "Change Focus", "Exit", "Deploy Application", "Uninstall Application"}
	for row, section := range keyList {
		table.SetCell(row, 0, keyCell(section))
		table.SetCell(row, 1, infoCell(descList[row]))
	}
	table.SetBackgroundColor(backgroundColor)
	return table
}
