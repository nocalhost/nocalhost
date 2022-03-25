/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package nocalhost

import (
	"fmt"
	"testing"
)

func TestGetAllKubeconfig(t *testing.T) {
	ks, err := GetAllKubeconfig()
	if err != nil {
		panic(err)
	}
	for _, k := range ks {
		fmt.Println(k)
	}
}
