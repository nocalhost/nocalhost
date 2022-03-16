/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import (
	"fmt"
	"github.com/rivo/tview"
	k8sruntime "k8s.io/apimachinery/pkg/util/runtime"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestRunTviewApplication(t *testing.T) {
	RunTviewApplication()
}

func TestSyncBuilder_Sync(t *testing.T) {

	app := tview.NewApplication()
	modal := tview.NewModal().
		SetText("Do you want to quit the application?").
		AddButtons([]string{"Quit", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Quit" {
				app.Stop()
			}
		})
	if err := app.SetRoot(modal, false).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

func TestDeleteLog(t *testing.T) {
	for i := 0; i < len(k8sruntime.ErrorHandlers); i++ {
		fn := runtime.FuncForPC(reflect.ValueOf(k8sruntime.ErrorHandlers[i]).Pointer()).Name()
		if strings.Contains(fn, "logError") {
			k8sruntime.ErrorHandlers = append(k8sruntime.ErrorHandlers[:i], k8sruntime.ErrorHandlers[i+1:]...)
		}
	}
	fmt.Printf("%v\n", k8sruntime.ErrorHandlers)
}
