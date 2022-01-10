/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"github.com/derailed/tview"
	"github.com/pkg/errors"
	"nocalhost/internal/nhctl/app_flags"
	"nocalhost/internal/nhctl/appmeta"
	"nocalhost/internal/nhctl/common"
	"nocalhost/pkg/nhctl/log"
	"os"
)

func (t *TviewApplication) buildDeployApplicationMenu() *tview.Table {
	menu := NewBorderedTable(" Deploy Application")
	menu.SetCell(0, 0, coloredCell(deployDemoAppOption))
	menu.SetCell(1, 0, coloredCell(deployHelmAppOption))
	menu.SetCell(2, 0, coloredCell(deployKubectlAppOption))
	menu.SetCell(3, 0, coloredCell(deployKustomizeAppOption))

	menu.SetSelectedFunc(func(row, column int) {
		selectedCell := menu.GetCell(row, column)
		var m tview.Primitive
		switch selectedCell.Text {
		case deployDemoAppOption:
			view := NewScrollingTextView(" Deploy BookInfo demo application")
			//nhctl install bookinfo --git-url https://github.com/nocalhost/bookinfo.git --type rawManifest --kubeconfig %s --namespace %s
			m = view
			f := app_flags.InstallFlags{
				GitUrl:  "https://github.com/nocalhost/bookinfo.git",
				AppType: string(appmeta.ManifestGit),
			}

			sbd := SyncBuilder{func(p []byte) (int, error) {
				t.app.QueueUpdateDraw(func() {
					view.Write([]byte(" " + string(p)))
				})
				return 0, nil
			}}

			log.RedirectionDefaultLogger(&sbd)
			go func() {
				_, err := common.InstallApplication(&f, "bookinfo", t.clusterInfo.KubeConfig, t.clusterInfo.NameSpace)
				if err != nil {
					panic(errors.Wrap(err, t.clusterInfo.KubeConfig))
				}
				log.RedirectionDefaultLogger(os.Stdout)
			}()
		default:
			return
		}
		t.switchBodyToC(menu, m)
	})
	return menu
}
