/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package log

import (
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestPWarn(t *testing.T) {
	PWarn("This is a warning")
	PWarnf("This is a warning in %d/%d/%d", 2021, 07, 25)
}

func TestES(t *testing.T) {
	InitEs("http://9.135.94.198:30001")
	writeStackToEs("debug", "ttt2222", "")
}

func TestGetIp(t *testing.T) {
	ip, err := externalIP()
	if err != nil {
		panic(err)
	}
	fmt.Println(ip.String())
}

func TestAddField(t *testing.T) {
	A()
	time.Sleep(5 * time.Second)
}

func A() {

	funcName, file, line, ok := runtime.Caller(2)
	go func() {
		if ok {
			fmt.Println("func name: " + runtime.FuncForPC(funcName).Name())
			fmt.Printf("file: %s, line: %d\n", file, line)
		}
	}()
}
