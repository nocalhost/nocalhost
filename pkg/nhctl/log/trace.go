/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package log

import "fmt"

// Trace TRACE logs do not print to stdout
func Trace(args ...interface{}) {
	writeStackToEs("TRACE", fmt.Sprintln(args...), "")
	if fileEntry != nil {
		fileEntry.Debug(args...)
	}
}

func Tracef(format string, args ...interface{}) {
	writeStackToEs("TRACE", fmt.Sprintf(format, args...), "")
	if fileEntry != nil {
		fileEntry.Debugf(format, args...)
	}
}
