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
