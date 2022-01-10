/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cluster

import (
	"fmt"
	"testing"
)

func TestName(t *testing.T) {
	cluster := NewVCluster("http://118.24.227.155")
	create, err := cluster.Create()
	fmt.Println(create, err)
	cluster.Delete()
}
