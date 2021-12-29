/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"github.com/derailed/tview"
	"github.com/gdamore/tcell/v2"
)

func NewBorderTable(s string) *tview.Table {
	t := tview.NewTable()
	t.SetBorder(true)
	t.SetTitle(s)
	t.SetSelectable(true, false)
	t.SetBackgroundColor(tcell.ColorBlack)
	return t
}
