/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
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
		l.rollingLog.MaxSize += 100 // add 100m
		n, _ = l.rollingLog.Write(p)
		return n, nil
	}
	return n, err
}
