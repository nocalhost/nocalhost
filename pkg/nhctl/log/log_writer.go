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

package log

import "gopkg.in/natefinch/lumberjack.v2"

type logWriter struct {
	rollingLog *lumberjack.Logger
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	if l == nil {
		return 0, nil
	}
	n, err = l.rollingLog.Write(p)
	if err != nil {
		l.rollingLog.MaxSize += 10 // add 10m
		return l.rollingLog.Write(p)
	}
	return n, err
}
