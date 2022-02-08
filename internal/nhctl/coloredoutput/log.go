/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package coloredoutput

import (
	"fmt"
	"io"
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

	writer io.Writer
)

func init() {
	if runtime.GOOS == "windows" {
		successSymbol = color.New(color.BgGreen, color.FgBlack).Sprint(" + ")
	}
	writer = color.Output
}

func SetWriter(w io.Writer) {
	writer = w
}

func ResetWriter() {
	writer = color.Output
}

// Yellow writes a line in yellow
func Yellow(format string, args ...interface{}) {
	fmt.Fprintln(writer, yellowString(format, args...))
}

// Green writes a line in green
func Green(format string, args ...interface{}) {
	fmt.Fprintln(writer, greenString(format, args...))
}

// BlueString returns a string in blue
func BlueString(format string, args ...interface{}) string {
	return blueString(format, args...)
}

// Success prints a message with the success symbol first, and the text in green
func Success(format string, args ...interface{}) {
	if writer == color.Output {
		fmt.Fprintf(writer, "%s %s\n", successSymbol, greenString(format, args...))
	} else if len(args) == 0 {
		fmt.Fprintln(writer, format)
	} else {
		fmt.Fprintf(writer, format, args)
	}
}

// Information prints a message with the information symbol first, and the text in blue
func Information(format string, args ...interface{}) {
	fmt.Fprintf(writer, "%s %s\n", informationSymbol, blueString(format, args...))
}

// Hint prints a message with the text in blue
func Hint(format string, args ...interface{}) {
	if writer == color.Output {
		fmt.Fprintf(writer, "%s\n", blueString(format, args...))
	} else if len(args) == 0 {
		fmt.Fprintln(writer, format)
	} else {
		fmt.Fprintf(writer, format, args)
	}
}

// Fail prints a message with the error symbol first, and the text in red
func Fail(format string, args ...interface{}) {
	fmt.Fprintf(writer, "%s %s\n", errorSymbol, redString(format, args...))
}

// Println writes a line with colors
func Println(args ...interface{}) {
	fmt.Fprintln(writer, args...)
}
