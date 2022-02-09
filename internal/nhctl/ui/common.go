/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"github.com/derailed/tview"
	"github.com/gdamore/tcell/v2"
	"go.uber.org/zap/zapcore"
	"nocalhost/internal/nhctl/coloredoutput"
	"nocalhost/internal/nhctl/nocalhost_path"
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"strings"
)

var (
	defaultKubeConfigPath = filepath.Join(utils.GetHomePath(), ".kube", "config")
	focusBorderColor      = tcell.ColorNavy
	backgroundColor       = tcell.Color(4294967528)
	textViewColor         = tcell.Color(4294967480)
	cliProfileDir         = filepath.Join(nocalhost_path.GetNhctlHomeDir(), "cli")
	cliProfileName        = filepath.Join(cliProfileDir, ".nocalhost_cli")
)

type CliProfile struct {
	LastPosition string `json:"lastPosition" json:"lastPosition"`
}

func NewRowSelectableTable(s string) *EnhancedTable {
	t := &EnhancedTable{
		Table: tview.NewTable(),
	}
	t.SetBorder(true)
	t.SetBorderPadding(0, 0, 1, 0)
	t.SetTitle(s)
	t.SetSelectable(true, false)
	t.SetBackgroundColor(backgroundColor)
	t.SetBorderFocusColor(focusBorderColor)
	return t
}

func NewSelectableTable(s string) *EnhancedTable {
	t := &EnhancedTable{
		Table: tview.NewTable(),
	}
	t.SetBorder(true)
	t.SetBorderPadding(0, 0, 1, 0)
	t.SetTitle(s)
	t.SetSelectable(true, true)
	t.SetBackgroundColor(backgroundColor)
	t.SetBorderFocusColor(focusBorderColor)
	return t
}

func NewTreeNode(s string) *tview.TreeNode {
	return tview.NewTreeNode(s).SetSelectable(true).SetColor(tcell.Color(4294967587))
}

func NewScrollingTextView(title string) *tview.TextView {
	t := tview.NewTextView()
	t.SetBorder(true)
	t.SetTitle(title)
	t.SetTextColor(textViewColor)
	t.SetBorderFocusColor(focusBorderColor)
	t.SetBackgroundColor(backgroundColor)
	return t
}

func trimSpaceStr(str string) string {
	return strings.Trim(str, " ")
}

func RedirectionOut(z zapcore.WriteSyncer) {
	log.RedirectionDefaultLogger(z)
	coloredoutput.SetWriter(z)
	clientgoutils.IoStreams.Out = z
	clientgoutils.IoStreams.ErrOut = z
}

func RecoverOut() {
	log.RedirectionDefaultLogger(os.Stdout)
	coloredoutput.ResetWriter()
	clientgoutils.IoStreams.Out = os.Stdout
	clientgoutils.IoStreams.ErrOut = os.Stdout
}

func LenOfFlex(f *tview.Flex) int {
	if f == nil {
		return 0
	}
	for i := 0; i < 100; i++ {
		it := f.ItemAt(i)
		if it == nil {
			return i
		}
	}
	return 100 // max len
}
