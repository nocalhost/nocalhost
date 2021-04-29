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

package coloredoutput

import (
	"fmt"
	"runtime"

	"github.com/fatih/color"
)

var (
	redString = color.New(color.FgHiRed).SprintfFunc()

	greenString = color.New(color.FgGreen).SprintfFunc()

	yellowString = color.New(color.FgHiYellow).SprintfFunc()

	blueString = color.New(color.FgHiBlue).SprintfFunc()

	errorSymbol = color.New(color.BgHiRed, color.FgBlack).Sprint(" x ")

	successSymbol = color.New(color.BgGreen, color.FgBlack).Sprint(" ✓ ")

	informationSymbol = color.New(color.BgHiBlue, color.FgBlack).Sprint(" i ")
)

func init() {
	if runtime.GOOS == "windows" {
		successSymbol = color.New(color.BgGreen, color.FgBlack).Sprint(" + ")
	}
}

// Yellow writes a line in yellow
func Yellow(format string, args ...interface{}) {
	fmt.Fprintln(color.Output, yellowString(format, args...))
}

// Green writes a line in green
func Green(format string, args ...interface{}) {
	fmt.Fprintln(color.Output, greenString(format, args...))
}

// BlueString returns a string in blue
func BlueString(format string, args ...interface{}) string {
	return blueString(format, args...)
}

// Success prints a message with the success symbol first, and the text in green
func Success(format string, args ...interface{}) {
	fmt.Fprintf(color.Output, "%s %s\n", successSymbol, greenString(format, args...))
}

// Information prints a message with the information symbol first, and the text in blue
func Information(format string, args ...interface{}) {
	fmt.Fprintf(color.Output, "%s %s\n", informationSymbol, blueString(format, args...))
}

// Hint prints a message with the text in blue
func Hint(format string, args ...interface{}) {
	fmt.Fprintf(color.Output, "%s\n", blueString(format, args...))
}

// Fail prints a message with the error symbol first, and the text in red
func Fail(format string, args ...interface{}) {
	fmt.Fprintf(color.Output, "%s %s\n", errorSymbol, redString(format, args...))
}

// Println writes a line with colors
func Println(args ...interface{}) {
	fmt.Fprintln(color.Output, args...)
}
