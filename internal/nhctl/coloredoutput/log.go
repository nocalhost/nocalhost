/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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

//type logger struct {
//	out  *logrus.Logger
//	file *logrus.Entry
//}
//
//var coloredoutput = &logger{
//	out: logrus.New(),
//}

func init() {
	if runtime.GOOS == "windows" {
		successSymbol = color.New(color.BgGreen, color.FgBlack).Sprint(" + ")
	}
}

// Init configures the logger for the package to use.
//func Init(level logrus.Level, dir, version string) {
//	coloredoutput.out.SetOutput(os.Stdout)
//	coloredoutput.out.SetLevel(level)
//
//	fileLogger := logrus.New()
//	fileLogger.SetFormatter(&logrus.TextFormatter{
//		DisableColors: true,
//		FullTimestamp: true,
//	})
//
//	logPath := filepath.Join(dir, "nocalhost.coloredoutput")
//	rolling := getRollingLog(logPath)
//	fileLogger.SetOutput(rolling)
//	fileLogger.SetLevel(logrus.DebugLevel)
//
//	actionID := uuid.New().String()
//	coloredoutput.file = fileLogger.WithFields(logrus.Fields{"action": actionID, "version": version})
//}

//func getRollingLog(path string) io.Writer {
//	return &lumberjack.Logger{
//		Filename:   path,
//		MaxSize:    1, // megabytes
//		MaxBackups: 10,
//		MaxAge:     28, //days
//		Compress:   true,
//	}
//}

// SetLevel sets the level of the main logger
//func SetLevel(level string) {
//	l, err := logrus.ParseLevel(level)
//	if err == nil {
//		coloredoutput.out.SetLevel(l)
//	}
//}
//
//// Debug writes a debug-level coloredoutput
//func Debug(args ...interface{}) {
//	coloredoutput.out.Debug(args...)
//	if coloredoutput.file != nil {
//		coloredoutput.file.Debug(args...)
//	}
//}
//
//// Debugf writes a debug-level coloredoutput with a format
//func Debugf(format string, args ...interface{}) {
//	coloredoutput.out.Debugf(format, args...)
//	if coloredoutput.file != nil {
//		coloredoutput.file.Debugf(format, args...)
//	}
//}
//
//// Info writes a info-level coloredoutput
//func Info(args ...interface{}) {
//	coloredoutput.out.Info(args...)
//	if coloredoutput.file != nil {
//		coloredoutput.file.Info(args...)
//	}
//}
//
//// Infof writes a info-level coloredoutput with a format
//func Infof(format string, args ...interface{}) {
//	coloredoutput.out.Infof(format, args...)
//	if coloredoutput.file != nil {
//		coloredoutput.file.Infof(format, args...)
//	}
//}
//
//// Error writes a error-level coloredoutput
//func Error(args ...interface{}) {
//	coloredoutput.out.Error(args...)
//	if coloredoutput.file != nil {
//		coloredoutput.file.Error(args...)
//	}
//}
//
//// Errorf writes a error-level coloredoutput with a format
//func Errorf(format string, args ...interface{}) {
//	coloredoutput.out.Errorf(format, args...)
//	if coloredoutput.file != nil {
//		coloredoutput.file.Errorf(format, args...)
//	}
//}
//
//// Fatalf writes a error-level coloredoutput with a format
//func Fatalf(format string, args ...interface{}) {
//	if coloredoutput.file != nil {
//		coloredoutput.file.Errorf(format, args...)
//	}
//
//	coloredoutput.out.Fatalf(format, args...)
//}

// Yellow writes a line in yellow
func Yellow(format string, args ...interface{}) {
	//log.out.Infof(format, args...)
	fmt.Fprintln(color.Output, yellowString(format, args...))
}

// Green writes a line in green
func Green(format string, args ...interface{}) {
	//log.out.Infof(format, args...)
	fmt.Fprintln(color.Output, greenString(format, args...))
}

// BlueString returns a string in blue
func BlueString(format string, args ...interface{}) string {
	return blueString(format, args...)
}

// Success prints a message with the success symbol first, and the text in green
func Success(format string, args ...interface{}) {
	//coloredoutput.out.Infof(format, args...)
	fmt.Fprintf(color.Output, "%s %s\n", successSymbol, greenString(format, args...))
}

// Information prints a message with the information symbol first, and the text in blue
func Information(format string, args ...interface{}) {
	//coloredoutput.out.Infof(format, args...)
	fmt.Fprintf(color.Output, "%s %s\n", informationSymbol, blueString(format, args...))
}

// Hint prints a message with the text in blue
func Hint(format string, args ...interface{}) {
	//coloredoutput.out.Infof(format, args...)
	fmt.Fprintf(color.Output, "%s\n", blueString(format, args...))
}

// Fail prints a message with the error symbol first, and the text in red
func Fail(format string, args ...interface{}) {
	//coloredoutput.out.Infof(format, args...)
	fmt.Fprintf(color.Output, "%s %s\n", errorSymbol, redString(format, args...))
}

// Println writes a line with colors
func Println(args ...interface{}) {
	//coloredoutput.out.Info(args...)
	fmt.Fprintln(color.Output, args...)
}
