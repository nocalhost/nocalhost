/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"github.com/derailed/tview"
	"nocalhost/internal/nhctl/utils"
	"path/filepath"
	"strings"
)

var (
	defaultKubeConfigPath = filepath.Join(utils.GetHomePath(), ".kube", "config")
)

func NewBorderedTable(s string) *tview.Table {
	t := tview.NewTable()
	t.SetBorder(true)
	t.SetTitle(s)
	t.SetSelectable(true, false)
	//t.SetBackgroundColor(tcell.ColorBlack)
	return t
}

func trimSpaceStr(str string) string {
	return strings.Trim(str, " ")
}
