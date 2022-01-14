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
	"nocalhost/internal/nhctl/utils"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"os"
	"path/filepath"
	"strings"
)

var (
	defaultKubeConfigPath = filepath.Join(utils.GetHomePath(), ".kube", "config")
	focusBorderColor      = tcell.ColorPaleGreen
)

const cliProfileName = ".nocalhost_cli"

type CliProfile struct {
	LastPosition string `json:"lastPosition" json:"lastPosition"`
}

func NewBorderedTable(s string) *EnhancedTable {
	t := &EnhancedTable{
		Table: tview.NewTable(),
	}
	t.SetBorder(true)
	t.SetBorderPadding(0, 0, 1, 0)
	t.SetTitle(s)
	t.SetSelectable(true, false)
	t.SetBorderFocusColor(focusBorderColor)
	return t
}

func NewScrollingTextView(title string) *tview.TextView {
	t := tview.NewTextView()
	t.SetBorder(true)
	t.SetTitle(title)
	t.SetBorderFocusColor(focusBorderColor)
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
