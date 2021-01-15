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
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	//"github.com/google/uuid"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var outLogger *zap.SugaredLogger
var fileEntry *zap.SugaredLogger
var logFile string

var fields = make(map[string]string, 0)
var core zapcore.Core

func init() {
	outLogger = getDefaultOutLogger() // if log is not be initiated explicitly (use log.Init()), the default out logger will be used.
	fields["PID"] = strconv.Itoa(os.Getpid())
	fields["PPID"] = strconv.Itoa(os.Getppid())
}

func getDefaultOutLogger() *zap.SugaredLogger {
	encoderConfig0 := zap.NewProductionEncoderConfig()
	encoderConfig0.EncodeTime = nil
	encoderConfig0.EncodeLevel = nil
	encoder2 := zapcore.NewConsoleEncoder(encoderConfig0)
	return zap.New(zapcore.NewCore(encoder2, zapcore.AddSync(os.Stdout), zap.InfoLevel)).Sugar()
}

func Init(level zapcore.Level, dir, fileName string) error {

	// stdout logger
	encoderConfig0 := zap.NewProductionEncoderConfig()
	encoderConfig0.EncodeTime = nil
	encoderConfig0.EncodeLevel = nil
	encoder2 := zapcore.NewConsoleEncoder(encoderConfig0)
	outLogger = zap.New(zapcore.NewCore(encoder2, zapcore.AddSync(os.Stdout), level)).Sugar()

	// file logger
	logPath := filepath.Join(dir, fileName)
	rolling := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    1, // megabytes
		MaxBackups: 10,
		MaxAge:     60, //days
		Compress:   true,
	}
	writeSyncer := zapcore.AddSync(rolling)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	//encoder := zapcore.NewConsoleEncoder(encoderConfig)
	encoder := zapcore.NewJSONEncoder(encoderConfig)
	core = zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)

	refreshFileLoggerWithFields()
	logFile = logPath
	return nil
}

func refreshFileLoggerWithFields() {
	args := make([]interface{}, 0)
	for key, val := range fields {
		args = append(args, key, val)
	}
	fileEntry = zap.New(core).Sugar().With(args...)
}

func AddField(key, val string) {
	fields[key] = val
	refreshFileLoggerWithFields()
}

func SetApp(a string) {
	app = a
}

func Debug(args ...interface{}) {
	outLogger.Debug(args...)
	if fileEntry != nil {
		fileEntry.Debug(args...)
	}
}

func Debugf(format string, args ...interface{}) {
	outLogger.Debugf(format, args...)
	if fileEntry != nil {
		fileEntry.Debugf(format, args...)
	}
}

func Info(args ...interface{}) {
	outLogger.Info(args...)
	if fileEntry != nil {
		fileEntry.Info(args)
	}
}

func Infof(format string, args ...interface{}) {
	outLogger.Infof(format, args...)
	if fileEntry != nil {
		fileEntry.Infof(format, args...)
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
		fileEntry.Warnf(format, args...)
	}
}

func WarnE(err error, message string) {
	if fileEntry != nil {
		fileEntry.Warnf("%s, err: %+v", message, err)
	}

	outLogger.Warn(fmt.Sprintf("[WARN] %s", message))
}

func Error(args ...interface{}) {
	outLogger.Error(args...)
	if fileEntry != nil {
		fileEntry.Error(args...)
	}
}

func Errorf(format string, args ...interface{}) {
	outLogger.Errorf(format, args...)
	if fileEntry != nil {
		fileEntry.Errorf(format, args...)
	}
}

func ErrorE(err error, message string) {
	if fileEntry != nil {
		fileEntry.Errorf("%s, err: %+v", message, err)
	}
	outLogger.Errorf("%s\nsee %s for more details", message, logFile)
}

func Fatal(args ...interface{}) {
	if fileEntry != nil {
		fileEntry.Error(args...)
	}
	outLogger.Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	if fileEntry != nil {
		fileEntry.Errorf(format, args...)
	}
	outLogger.Fatalf(format, args...)
}

// log with error
func FatalE(err error, message string) {

	outLogger.Errorf("%s\nsee %s for more details", message, logFile)

	if fileEntry != nil {
		fileEntry.Fatalf("%s, err: %+v", message, err)
	}
}
