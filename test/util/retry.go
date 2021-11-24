/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package util

import (
	"fmt"
	"nocalhost/pkg/nhctl/clientgoutils"
	"nocalhost/pkg/nhctl/log"
	"reflect"
	"runtime"
	"strings"
	"time"
)

var retryTimes = 10

func Retry(suiteName string, funcs []func() error) {

	logger := log.TestLogger(suiteName)
	var err error
	for i, f := range funcs {
		for i := 0; i < retryTimes; i++ {
			if err = f(); err == nil {
				break
			}
			logger.Infof("Error while exec retry func, %v", err)
		}
		clientgoutils.MustI(
			err, fmt.Sprintf(
				"error on step [%d] %s - %s",
				i, suiteName, getFunctionName(reflect.ValueOf(f)),
			),
		)
	}
}

func RetryFunc(fun func() error) error {
	var err error
	for i := 0; i < retryTimes; i++ {
		time.Sleep(1 * time.Second)
		if err = fun(); err == nil {
			break
		}
	}
	return err
}

func getFunctionName(f reflect.Value) string {
	fn := runtime.FuncForPC(f.Pointer()).Name()
	return fn[strings.LastIndex(fn, ".")+1:]
}
