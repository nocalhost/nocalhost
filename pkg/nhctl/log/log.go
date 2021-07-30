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

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	//"github.com/google/uuid"
	lumberjack "gopkg.in/natefinch/lumberjack.v2"
)

var stdoutLogger *zap.SugaredLogger
var stderrLogger *zap.SugaredLogger
var fileEntry *zap.SugaredLogger
var logFile string

var fields = make(map[string]string, 0)
var core zapcore.Core

func init() {
	// if log is not be initiated explicitly (use log.Init()),
	// the default out logger will be used.
	stdoutLogger = getDefaultOutLogger(os.Stdout)
	stderrLogger = getDefaultOutLogger(os.Stderr)
	fields["PID"] = strconv.Itoa(os.Getpid())
	fields["PPID"] = strconv.Itoa(os.Getppid())
}

func getDefaultOutLogger(w zapcore.WriteSyncer) *zap.SugaredLogger {
	encoderConfig0 := zap.NewProductionEncoderConfig()
	encoderConfig0.EncodeTime = nil
	encoderConfig0.EncodeLevel = nil
	encoder2 := zapcore.NewConsoleEncoder(encoderConfig0)
	return zap.New(zapcore.NewCore(encoder2, zapcore.AddSync(os.Stdout), zap.InfoLevel), zap.ErrorOutput(w)).Sugar()
}

func Init(level zapcore.Level, dir, fileName string) error {

	// stdout logger
	encoderConfig0 := zap.NewProductionEncoderConfig()
	encoderConfig0.EncodeTime = nil
	encoderConfig0.EncodeLevel = nil
	encoder2 := zapcore.NewConsoleEncoder(encoderConfig0)
	stdoutLogger = zap.New(zapcore.NewCore(encoder2, zapcore.AddSync(os.Stdout), level)).Sugar()
	stderrLogger = zap.New(zapcore.NewCore(encoder2, zapcore.AddSync(os.Stderr), level)).Sugar()

	// file logger
	logPath := filepath.Join(dir, fileName)
	rolling := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    50, // megabytes
		MaxBackups: 60,
		MaxAge:     60, //days
		Compress:   true,
	}
	writeSyncer := zapcore.AddSync(rolling)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = CustomTimeEncoder
	encoderConfig.EncodeLevel = CustomLevelEncoder
	encoderConfig.EncodeDuration = CustomDurationEncoder

	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	core = zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)

	refreshFileLoggerWithFields()
	logFile = logPath
	return nil
}

func CustomDurationEncoder(t time.Duration, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.String())
}

func CustomTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format(time.Stamp))
}

func CustomLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
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

func Debug(args ...interface{}) {
	stdoutLogger.Debug(args...)
	if fileEntry != nil {
		fileEntry.Debug(args...)
	}
}

func Debugf(format string, args ...interface{}) {
	stdoutLogger.Debugf(format, args...)
	if fileEntry != nil {
		fileEntry.Debugf(format, args...)
	}
}

func Info(args ...interface{}) {
	stdoutLogger.Info(args...)
	if fileEntry != nil {
		fileEntry.Info(args...)
	}
}

func Infof(format string, args ...interface{}) {
	stdoutLogger.Infof(format, args...)
	if fileEntry != nil {
		fileEntry.Infof(format, args...)
	}
}

func Warn(args ...interface{}) {
	stdoutLogger.Warn(args...)
	if fileEntry != nil {
		fileEntry.Warn(args...)
	}
}

func Warnf(format string, args ...interface{}) {
	stdoutLogger.Warnf(format, args...)
	if fileEntry != nil {
		fileEntry.Warnf(format, args...)
	}
}

func WarnE(err error, message string) {
	if fileEntry != nil {
		fileEntry.Warnf("%s, err: %+v", message, err)
	}

	if err != nil {
		stdoutLogger.Warn(fmt.Sprintf("%s: %s", message, err.Error()))
	} else {
		stdoutLogger.Warn(fmt.Sprintf("%s", message))
	}
}

func Error(args ...interface{}) {
	stdoutLogger.Error(args...)
	if fileEntry != nil {
		fileEntry.Error(args...)
	}
}

func Errorf(format string, args ...interface{}) {
	stdoutLogger.Errorf(format, args...)
	if fileEntry != nil {
		fileEntry.Errorf(format, args...)
	}
}

func ErrorE(err error, message string) {
	if fileEntry != nil {
		fileEntry.Errorf("%s, err: %+v", message, err)
	}
	if err != nil {
		stdoutLogger.Errorf("%s: %s", message, err.Error())
	} else {
		stdoutLogger.Errorf("%s", message)
	}
}

func Fatal(args ...interface{}) {
	if fileEntry != nil {
		fileEntry.Error(args...)
	}
	stderrLogger.Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	if fileEntry != nil {
		fileEntry.Errorf(format, args...)
	}
	stderrLogger.Fatalf(format, args...)
}

// log with error
func FatalE(err error, message string) {

	if err != nil {
		stderrLogger.Errorf("%s: %s", message, err.Error())
	} else {
		stderrLogger.Errorf("%s", message)
	}

	if fileEntry != nil {
		fileEntry.Fatalf("%s, err: %+v", message, err)
	}
}

func LogE(err error) {
	if fileEntry != nil {
		fileEntry.Errorf("%+v", err)
	}
}

func Log(args ...interface{}) {
	if fileEntry != nil {
		fileEntry.Info(args...)
	}
}

func Logf(format string, args ...interface{}) {
	if fileEntry != nil {
		fileEntry.Infof(format, args...)
	}
}

func TLogf(tag, format string, args ...interface{}) {
	if fileEntry != nil {
		fileEntry.With("tag", tag).Infof(format, args...)
	}
}

func LogStack() {
	if fileEntry != nil {
		fileEntry.Debug(string(debug.Stack()))
	}
}

// For IDE Plugin

func PWarn(info string) {
	stdoutLogger.Info("[WARNING] " + info)
	if fileEntry != nil {
		fileEntry.Warn(info)
	}
}

func PWarnf(format string, args ...interface{}) {
	stdoutLogger.Warnf("[WARNING] "+format, args...)
	if fileEntry != nil {
		fileEntry.Warnf(format, args...)
	}
}

func PInfo(info string) {
	stdoutLogger.Info("[INFO] " + info)
	if fileEntry != nil {
		fileEntry.Info(info)
	}
}

func PInfof(format string, args ...interface{}) {
	stdoutLogger.Infof("[INFO] "+format, args...)
	if fileEntry != nil {
		fileEntry.Infof(format, args...)
	}
}

func PError(info string) {
	stdoutLogger.Info("[ERROR] " + info)
	if fileEntry != nil {
		fileEntry.Info(info)
	}
}

func PErrorf(format string, args ...interface{}) {
	stdoutLogger.Infof("[ERROR] "+format, args...)
	if fileEntry != nil {
		fileEntry.Errorf(format, args...)
	}
}
