/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"nocalhost/internal/nhctl/profile"
	"nocalhost/internal/nhctl/utils"
	"testing"
	"time"
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

//func TestDeploymentController_ReplaceImage(t *testing.T) {
//	client, err := clientgoutils.NewClientGoUtils("", "nocalhost-test")
//	objs, err := client.Get("deployment", "reviews")
//	if err != nil {
//		panic(err)
//	}
//
//	obj := &unstructured.Unstructured{}
//	obj.Object, err = runtime.DefaultUnstructuredConverter.ToUnstructured(objs[0])
//	if err != nil {
//		panic(err)
//	}
//
//	podSpec, err := GetPodTemplateFromSpecPath("/spec/template/spec", obj.Object)
//	if err != nil {
//		panic(err)
//	}
//
//	fmt.Printf("%v\n", podSpec)
//}

func TestPrintForPod(t *testing.T) {
	ctx := context.TODO()
	content := make(chan string, 0)

	newSpinner := utils.NewSpinner("zzzzzzz")
	newSpinner.Start()

	breaking := false

	for {
		select {
		case <-ctx.Done():
			newSpinner.Stop()
			breaking = true
		case c := <-content:
			newSpinner.Update(c)
		}

		if breaking {
			break
		}
	}

	time.Sleep(time.Minute)
}
