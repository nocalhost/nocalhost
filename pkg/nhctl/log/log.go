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

package log

import (
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

//var log = logrus.New()
var outLogger  *logrus.Logger
var fileEntry *logrus.Entry

func init() {
	//log.SetFormatter(&nested.Formatter{
	//	HideKeys: true,
	//	//FieldsOrder: []string{"component", "category"},
	//})
	outLogger = logrus.New()
}

func Init(level logrus.Level, dir, fileName string) {
	outLogger.SetOutput(os.Stdout)
	outLogger.SetLevel(level)

	fileLogger := logrus.New()
	fileLogger.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})

	logPath := filepath.Join(dir, fileName)
	rolling := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    1, // megabytes
		MaxBackups: 10,
		MaxAge:     60, //days
		Compress:   true,
	}
	fileLogger.SetOutput(rolling)
	fileLogger.SetLevel(logrus.DebugLevel)

	traceId := uuid.New().String()
	fileEntry = fileLogger.WithFields(logrus.Fields{"traceId": traceId})
}

func SetLevel(level logrus.Level) {
	outLogger.SetLevel(level)
}

func Fatalf(format string, args ...interface{}) {
	outLogger.Fatalf(format, args...)
	if fileEntry != nil {
		fileEntry.Fatalf(format,args...)
	}
}

func Fatal(args ...interface{}) {
	outLogger.Fatal(args...)
	if fileEntry != nil {
		fileEntry.Fatal(args...)
	}
}

func Warn(args ...interface{}) {
	outLogger.Warn(args...)
	if fileEntry != nil {
		fileEntry.Warn(args...)
	}
}

func Warnf(format string, args ...interface{}) {
	outLogger.Warnf(format, args...)
	if fileEntry != nil {
		fileEntry.Warnf(format,args...)
	}
}

func Debugf(format string, args ...interface{}) {
	outLogger.Debugf(format, args...)
	if fileEntry != nil {
		fileEntry.Debugf(format,args...)
	}
}

func Debug(args ...interface{}) {
	outLogger.Debug(args...)
	if fileEntry != nil {
		fileEntry.Debug(args...)
	}
}

func Info(args ...interface{}) {
	outLogger.Info(args...)
	if fileEntry != nil {
		fileEntry.Info(args...)
	}
}

func Infof(format string, args ...interface{}) {
	outLogger.Infof(format, args...)
	if fileEntry != nil {
		fileEntry.Infof(format, args...)
	}
}


