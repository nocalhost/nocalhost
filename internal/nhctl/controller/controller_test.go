/*
* Copyright (C) 2020 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package controller

import (
	"encoding/json"
	"fmt"
	"nocalhost/internal/nhctl/profile"
	"testing"
)

func TestIsResourcesLimitToLow(t *testing.T) {
	r := &profile.ResourceQuota{
		Limits:   &profile.QuotaList{Memory: "1.5Gi", Cpu: "1"},
		Requests: &profile.QuotaList{Memory: "50Mi", Cpu: "100m"},
	}
	rq, _ := convertResourceQuota(r)
	fmt.Println(rq.Limits.Cpu().String())
	//bys, _ := json.Marshal(rq)
	//fmt.Printf("%v\n", string(bys))
	bys, _ := json.Marshal(rq.Limits)
	fmt.Println(string(bys))
	fmt.Println(IsResourcesLimitTooLow(rq))
	fmt.Println(IsResourcesLimitTooLow(nil))
	r.Limits = nil
	rq, _ = convertResourceQuota(r)
	fmt.Println(IsResourcesLimitTooLow(rq))
}
