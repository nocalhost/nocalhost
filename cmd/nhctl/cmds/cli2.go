/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
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
	pageCount := 10
	app := tview.NewApplication()
	list := tview.NewList().
		AddItem("List item 1", "Some explanatory text", 'a', nil).
		AddItem("List item 2", "Some explanatory text", 'b', nil).
		AddItem("List item 3", "Some explanatory text", 'c', nil).
		AddItem("List item 4", "Some explanatory text", 'd', nil).
		AddItem("Quit", "Press to exit", 'q', func() {
			app.Stop()
		})
	pages := tview.NewPages()
	pages.AddPage("aaa", list, false, true)
	for page := 0; page < pageCount; page++ {
		func(page int) {
			pages.AddPage(fmt.Sprintf("page-%d", page),
				tview.NewModal().
					SetText(fmt.Sprintf("This is page %d. Choose where to go next.", page+1)).
					AddButtons([]string{"Next", "Quit"}).
					SetDoneFunc(func(buttonIndex int, buttonLabel string) {
						if buttonIndex == 0 {
							pages.SwitchToPage(fmt.Sprintf("page-%d", (page+1)%pageCount))
						} else {
							app.Stop()
						}
					}),
				false,
				page == 0)
		}(page)
	}
	if err := app.SetRoot(pages, true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
