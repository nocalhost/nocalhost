/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package cmds

import (
	"fmt"
	"regexp"
	"testing"
)

func TestReg(t *testing.T) {
	pattern := "^v\\d+\\.\\d+\\.\\d+"
	match, _ := regexp.MatchString(pattern, "v0.1.1")
	if !match {
		fmt.Printf("nocalhost-api version %s is invalid, no matching: %v\n", "aaaa", `^v\\d+\\.\\d+\\.\\d+`)
	}

	r, _ := regexp.Compile("\\d+")
	nums := r.FindAllString("v0.0.123.134-123", -1)
	for _, num := range nums {
		fmt.Println(num)
	}

	nums2 := r.FindAllString("v0.0.123.134-123", -1)
	for _, num := range nums2 {
		fmt.Println(num)
	}
}

func TestCompareVersion(t *testing.T) {
	fmt.Println(CompareVersion("v0.0.1", "v0.0.2"))
	fmt.Println(CompareVersion("v0.1.1", "v0.0.2"))
	fmt.Println(CompareVersion("v0.1.1", "v0.1.1"))
	fmt.Println(CompareVersion("v0.1.1-rc1", "v0.1.1"))
	fmt.Println(CompareVersion("v0.1.1-rc1", "v0.1.1-rc2"))
}