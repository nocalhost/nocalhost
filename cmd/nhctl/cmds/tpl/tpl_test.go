/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package tpl

import (
	"fmt"
	"testing"
)

func TestGetSvcTpl(t *testing.T) {
	tpl, err := GetSvcTpl("aa")
	if err != nil {
		panic(err)
	}
	fmt.Println(tpl)
}
