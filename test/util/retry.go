/*
 * Tencent is pleased to support the open source community by making Nocalhost available.,
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under,
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
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
	var err error
	for i, f := range funcs {
		for i := 0; i < retryTimes; i++ {
			if err = f(); err == nil {
				break
			}
			log.Info(err)
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
