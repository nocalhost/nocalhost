/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

//func (t *TviewApplication) buildDeployApplicationMenu(ns string) *EnhancedTable {
//	menu := t.NewBorderedTable("Deploy Application")
//	menu.SetCell(0, 0, coloredCell(deployDemoAppOption))
//	menu.SetCell(1, 0, coloredCell(deployHelmAppOption))
//	menu.SetCell(2, 0, coloredCell(deployKubectlAppOption))
//	menu.SetCell(3, 0, coloredCell(deployKustomizeAppOption))
//
//	menu.SetSelectedFunc(func(row, column int) {
//		selectedCell := menu.GetCell(row, column)
//		switch selectedCell.Text {
//		case deployDemoAppOption:
//			t.app.QueueEvent(tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone))
//			sbd := t.switchBodyToScrollingView("", nil)
//			//nhctl install bookinfo --git-url https://github.com/nocalhost/bookinfo.git --type rawManifest --kubeconfig %s --namespace %s
//			f := app_flags.InstallFlags{
//				GitUrl:  "https://github.com/nocalhost/bookinfo.git",
//				AppType: string(appmeta.ManifestGit),
//			}
//			log.RedirectionDefaultLogger(sbd)
//			go func() {
//				_, err := common.InstallApplication(&f, "bookinfo", t.clusterInfo.KubeConfig, ns)
//				if err != nil {
//					t.ShowInfo(err.Error())
//				}
//				log.RedirectionDefaultLogger(os.Stdout)
//			}()
//		default:
//			return
//		}
//	})
//	return menu
//}
