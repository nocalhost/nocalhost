/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package ui

import "github.com/derailed/tview"

type EnhancedTable struct {
	*tview.Table
	blurFunc  func()
	focusFunc func()
}

func (e *EnhancedTable) SetBlurFunc(f func()) {
	e.blurFunc = f
}

func (e *EnhancedTable) SetFocusFunc(f func()) {
	e.focusFunc = f
}

func (e *EnhancedTable) Blur() {
	if e.blurFunc != nil {
		go func() {
			e.blurFunc()
		}()
	}
	e.Table.Blur()
}

func (e *EnhancedTable) Focus(delegate func(p tview.Primitive)) {
	if e.focusFunc != nil {
		e.focusFunc()
	}
	e.Table.Focus(delegate)
}
