/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(cli2Command)
}

var cli2Command = &cobra.Command{
	Use:   "cli2",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		RunTviewApplication()
	},
}

func RunTviewApplication() {
	app := tview.NewApplication()
	pages := tview.NewPages()
	main := tview.NewFlex().SetDirection(tview.FlexRow)
	table := tview.NewTable()
	table.SetTitle("Menu")
	table.SetBorder(true)
	table.SetCellSimple(0, 0, " Deploy Application")
	table.SetCellSimple(1, 0, " List Workloads")
	table.SetCellSimple(2, 0, " Start DevMode")
	table.SetBackgroundColor(tcell.ColorBlack)
	table.GetCell(0, 0).SetTextColor(tcell.ColorPaleGreen)
	table.GetCell(1, 0).SetTextColor(tcell.ColorPaleGreen)
	table.GetCell(2, 0).SetTextColor(tcell.ColorPaleGreen)

	table.SetSelectedFunc(func(row, column int) {
		table.GetCell(row, column).SetTextColor(tcell.ColorRed)
	})
	table.SetSelectable(true, false)
	main.AddItem(table, 0, 1, false)
	main.AddItem(tview.NewTextView().SetText("hello world"), 1, 1, true)
	pages.AddPage("Main", main, true, true)
	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
