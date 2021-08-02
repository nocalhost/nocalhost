/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package utils

import (
	"strconv"
	"time"
)

// GetDate
func GetDate() string {
	return time.Now().Format("2006/01/02")
}

// GetTodayDateInt
func GetTodayDateInt() int {
	dateStr := time.Now().Format("200601")
	date, err := strconv.Atoi(dateStr)
	if err != nil {
		return 0
	}
	return date
}

// TimeLayout
func TimeLayout() string {
	return "2006-01-02 15:04:05"
}

// TimeToString
func TimeToString(ts time.Time) string {
	return time.Unix(ts.Unix(), 00).Format(TimeLayout())
}

// TimeToShortString
func TimeToShortString(ts time.Time) string {
	return time.Unix(ts.Unix(), 00).Format("2006.01.02")
}
