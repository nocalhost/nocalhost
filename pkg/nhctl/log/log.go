/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
 */

package log

import (
	"fmt"
	"github.com/pkg/errors"
	_const "nocalhost/internal/nhctl/const"
	"os"
	"path/filepath"
	"runtime"
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

var fields = make(map[string]string, 0)
var fileLogsConfig zapcore.Core

func init() {
	// if log is not be initiated explicitly (use log.Init()),
	// the default out logger will be used.
	stdoutLogger = getDefaultOutLogger(os.Stdout)
	stderrLogger = getDefaultOutLogger(os.Stderr)
	fields["PID"] = strconv.Itoa(os.Getpid())
	fields["PPID"] = strconv.Itoa(os.Getppid())
}

func RedirectionDefaultLogger(w zapcore.WriteSyncer) {
	stdoutLogger = getDefaultOutLogger(w)
}

func GetLogger(w zapcore.WriteSyncer) *zap.SugaredLogger {
	return getDefaultOutLogger(w)
}

func getDefaultOutLogger(w zapcore.WriteSyncer) *zap.SugaredLogger {
	encoderConfig0 := zap.NewProductionEncoderConfig()
	encoderConfig0.EncodeTime = nil
	encoderConfig0.EncodeLevel = nil
	encoder2 := zapcore.NewConsoleEncoder(encoderConfig0)
	return zap.New(zapcore.NewCore(encoder2, zapcore.AddSync(w), zap.InfoLevel), zap.ErrorOutput(w)).Sugar()
}

func Init(level zapcore.Level, dir, fileName string) error {

	// stdout logger cfg
	cfg := zap.NewProductionEncoderConfig()
	if fullLog() {
		cfg.EncodeTime = CustomTimeEncoder
		cfg.EncodeLevel = CustomLevelEncoder
	} else {
		cfg.EncodeTime = nil
		cfg.EncodeLevel = nil
	}

	unFormatEncoder := zapcore.NewConsoleEncoder(cfg)
	unFormatStdoutConfig := zapcore.NewCore(unFormatEncoder, zapcore.AddSync(os.Stdout), level)
	unFormatStderrConfig := zapcore.NewCore(unFormatEncoder, zapcore.AddSync(os.Stderr), level)

	// file logger cfg
	logPath := filepath.Join(dir, fileName)
	rolling := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    100, // megabytes
		MaxBackups: 60,
		MaxAge:     120, //days
		Compress:   true,
	}
	rollingLog := &logWriter{rollingLog: rolling}
	writeSyncer := zapcore.AddSync(rollingLog)
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = CustomTimeEncoder
	encoderConfig.EncodeLevel = CustomLevelEncoder
	encoderConfig.EncodeDuration = CustomDurationEncoder

	encoder := zapcore.NewConsoleEncoder(encoderConfig)
	fileLogsConfig = zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)

	// init
	initOrReInitStdout(unFormatStdoutConfig)
	initOrReInitStderr(unFormatStderrConfig)
	initOrReInitFileEntry(fileLogsConfig, fields)
	return nil
}

func fullLog() bool {
	return os.Getenv(_const.EnableFullLogEnvKey) != ""
}

func initOrReInitStdout(configuration zapcore.Core) {
	stdoutLogger = zap.New(configuration).Sugar()
}

func initOrReInitStderr(configuration zapcore.Core) {
	stderrLogger = zap.New(configuration).Sugar()
}

func initOrReInitFileEntry(configuration zapcore.Core, fields map[string]string) {
	args := make([]interface{}, 0)
	for key, val := range fields {
		args = append(args, key, val)
	}
	fileEntry = zap.New(configuration).Sugar().With(args...)
}

func AddField(key, val string) {
	fields[key] = val
	initOrReInitFileEntry(fileLogsConfig, fields)
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

func WriteToEsWithField(field map[string]interface{}, format string, args ...interface{}) {
	writeStackToEsWithField("INFO", fmt.Sprintf(format, args...), "", field)
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Debugf(format, args...)
	}
}

func Debug(args ...interface{}) {
	writeStackToEs("DEBUG", fmt.Sprintln(args...), "")
	stdoutLogger.Debug(args...)
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Debug(args...)
	}
}

func Debugf(format string, args ...interface{}) {
	writeStackToEs("DEBUG", fmt.Sprintf(format, args...), "")
	stdoutLogger.Debugf(format, args...)
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Debugf(format, args...)
	}
}

func Info(args ...interface{}) {
	writeStackToEs("INFO", fmt.Sprintln(args...), "")
	stdoutLogger.Info(args...)
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Info(args...)
	}
}

func Infof(format string, args ...interface{}) {
	writeStackToEs("INFO", fmt.Sprintf(format, args...), "")
	stdoutLogger.Infof(format, args...)
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Infof(format, args...)
	}
}

func Warn(args ...interface{}) {
	writeStackToEs("WARN", fmt.Sprintln(args...), "")
	stdoutLogger.Warn(args...)
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Warn(args...)
	}
}

func Warnf(format string, args ...interface{}) {
	writeStackToEs("WARN", fmt.Sprintf(format, args...), "")
	stdoutLogger.Warnf(format, args...)
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Warnf(format, args...)
	}
}

func WarnE(err error, message string) {
	writeStackToEs("WARN", message, fmt.Sprintf("%+v", err))

	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Warnf("%s, err: %+v", message, err)
	}

	if err != nil {
		if message != "" {
			stdoutLogger.Warnf("%s: %s", message, err.Error())
		} else {
			stdoutLogger.Warn(err.Error())
		}
	} else {
		stdoutLogger.Warn(fmt.Sprintf("%s", message))
	}
}

func Error(args ...interface{}) {
	writeStackToEs("ERROR", fmt.Sprintln(args...), "")
	stdoutLogger.Error(args...)
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Error(args...)
	}
}

func Errorf(format string, args ...interface{}) {
	writeStackToEs("ERROR", fmt.Sprintf(format, args...), "")
	stdoutLogger.Errorf(format, args...)
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Errorf(format, args...)
	}
}

func ErrorE(err error, message string) {
	writeStackToEs("ERROR", message, fmt.Sprintf("%+v", err))
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Errorf("%s, err: %+v", message, err)
	}
	if err != nil {
		if message != "" {
			stdoutLogger.Errorf("%s: %s", message, err.Error())
		} else {
			stdoutLogger.Error(err.Error())
		}
	} else {
		stdoutLogger.Errorf("%s", message)
	}
}

func Fatal(args ...interface{}) {
	writeStackToEs("FATAL", fmt.Sprintln(args...), "")
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Error(args...)
	}
	stderrLogger.Fatal(args...)
}

func Fatalf(format string, args ...interface{}) {
	writeStackToEs("FATAL", fmt.Sprintf(format, args...), "")
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Errorf(format, args...)
	}
	stderrLogger.Fatalf(format, args...)
}

// log with error
func FatalE(err error, message string) {
	writeStackToEs("FATAL", message, fmt.Sprintf("%+v", err))
	if err != nil {
		if message != "" {
			stderrLogger.Errorf("%s: %s", message, err.Error())
		} else {
			stderrLogger.Errorf("%s", err.Error())
		}
	} else {
		stderrLogger.Errorf("%s", message)
	}

	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Fatalf("%s, err: %+v", message, err)
	}
}

func WrapAndLogE(err error) {
	if err != nil {
		return
	}
	LogE(errors.Wrap(err, ""))
}

func LogE(err error) {
	if err == nil {
		return
	}
	writeStackToEs("LOG", err.Error(), fmt.Sprintf("%+v", err))
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Errorf("%+v", err)
	}
}

func Log(args ...interface{}) {
	writeStackToEs("LOG", fmt.Sprintln(args...), "")
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Info(args...)
	}
}

func Logf(format string, args ...interface{}) {
	writeStackToEs("LOG", fmt.Sprintf(format, args...), "")
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Infof(format, args...)
	}
}

func LogDebugf(format string, args ...interface{}) {
	writeStackToEs("DEBUG", fmt.Sprintf(format, args...), "")
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Debugf(format, args...)
	}
}

func TLogf(tag, format string, args ...interface{}) {
	writeStackToEs("TLOG", fmt.Sprintf(format, args...), "")
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).With("tag", tag).Infof(format, args...)
	}
}

func LogStack() {
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Debug(string(debug.Stack()))
	}
}

// For IDE Plugin

func PWarn(info string) {
	writeStackToEs("[WARNING]", fmt.Sprintln(info), "")
	stdoutLogger.Info("[WARNING] " + info)
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Warn(info)
	}
}

func PWarnf(format string, args ...interface{}) {
	writeStackToEs("[WARNING]", fmt.Sprintf(format, args...), "")
	stdoutLogger.Warnf("[WARNING] "+format, args...)
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Warnf(format, args...)
	}
}

func PInfo(info string) {
	stdoutLogger.Info("[INFO] " + info)
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Info(info)
	}
}

func PInfof(format string, args ...interface{}) {
	stdoutLogger.Infof("[INFO] "+format, args...)
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Infof(format, args...)
	}
}

func PError(info string) {
	stdoutLogger.Info("[ERROR] " + info)
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Info(info)
	}
}

func PErrorf(format string, args ...interface{}) {
	stdoutLogger.Infof("[ERROR] "+format, args...)
	if fileEntry != nil {
		_, fn, line, _ := runtime.Caller(1)
		fileEntry.With("fn", fn, "line", line).Errorf(format, args...)
	}
}
