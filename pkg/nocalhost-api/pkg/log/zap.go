/*
* Copyright (C) 2021 THL A29 Limited, a Tencent company.  All rights reserved.
* This source code is licensed under the Apache License Version 2.0.
*/

package log

import (
	"io"
	"os"
	"strings"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"nocalhost/pkg/nocalhost-api/pkg/utils"
)

const (
	// WriterStdOut
	WriterStdOut = "stdout"
	// WriterFile
	WriterFile = "file"
)

const (
	// RotateTimeDaily
	RotateTimeDaily = "daily"
	// RotateTimeHourly
	RotateTimeHourly = "hourly"
)

// zapLogger logger struct
type zapLogger struct {
	sugaredLogger *zap.SugaredLogger
}

// newZapLogger new zap logger
func newZapLogger(cfg *Config) (Logger, error) {
	encoder := getJSONEncoder()

	var cores []zapcore.Core
	var options []zap.Option
	option := zap.Fields(zap.String("ip", utils.GetLocalIP()), zap.String("app", viper.GetString("name")))
	options = append(options, option)

	allLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl <= zapcore.FatalLevel
	})

	writers := strings.Split(cfg.Writers, ",")
	for _, w := range writers {
		if w == WriterStdOut {
			core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)
			cores = append(cores, core)
		}
		if w == WriterFile {
			infoFilename := cfg.LoggerFile
			infoWrite := getLogWriterWithTime(infoFilename)
			warnFilename := cfg.LoggerWarnFile
			warnWrite := getLogWriterWithTime(warnFilename)
			errorFilename := cfg.LoggerErrorFile
			errorWrite := getLogWriterWithTime(errorFilename)

			infoLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl <= zapcore.InfoLevel
			})
			warnLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				stacktrace := zap.AddStacktrace(zapcore.WarnLevel)
				options = append(options, stacktrace)
				return lvl == zapcore.WarnLevel
			})
			errorLevel := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				stacktrace := zap.AddStacktrace(zapcore.ErrorLevel)
				options = append(options, stacktrace)
				return lvl >= zapcore.ErrorLevel
			})

			core := zapcore.NewCore(encoder, zapcore.AddSync(infoWrite), infoLevel)
			cores = append(cores, core)
			core = zapcore.NewCore(encoder, zapcore.AddSync(warnWrite), warnLevel)
			cores = append(cores, core)
			core = zapcore.NewCore(encoder, zapcore.AddSync(errorWrite), errorLevel)
			cores = append(cores, core)
		}
		if w != WriterFile && w != WriterStdOut {
			core := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)
			cores = append(cores, core)
			allWriter := getLogWriterWithTime(cfg.LoggerFile)
			core = zapcore.NewCore(encoder, zapcore.AddSync(allWriter), allLevel)
			cores = append(cores, core)
		}
	}

	combinedCore := zapcore.NewTee(cores...)

	// debug
	caller := zap.AddCaller()
	options = append(options, caller)
	development := zap.Development()
	options = append(options, development)
	addCallerSkip := zap.AddCallerSkip(2)
	options = append(options, addCallerSkip)

	logger := zap.New(combinedCore, options...).Sugar()

	return &zapLogger{sugaredLogger: logger}, nil
}

// getJSONEncoder
func getJSONEncoder() zapcore.Encoder {
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "time",
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		NameKey:        "app",
		CallerKey:      "file",
		StacktraceKey:  "trace",
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
	}
	return zapcore.NewJSONEncoder(encoderConfig)
}

// getLogWriterWithTime
func getLogWriterWithTime(filename string) io.Writer {
	logFullPath := filename
	rotationPolicy := viper.Get("coloredoutput.log_rolling_policy")
	backupCount := viper.GetUint("coloredoutput.log_backup_count")
	// 默认
	rotateDuration := time.Hour * 24
	if rotationPolicy == RotateTimeHourly {
		rotateDuration = time.Hour
	}
	hook, err := rotatelogs.New(
		logFullPath+".%Y%m%d%H",
		rotatelogs.WithLinkName(logFullPath),
		rotatelogs.WithRotationCount(backupCount),
		rotatelogs.WithRotationTime(rotateDuration),
	)

	if err != nil {
		panic(err)
	}
	return hook
}

// Debug logger
func (l *zapLogger) Debug(args ...interface{}) {
	l.sugaredLogger.Debug(args...)
}

// Info logger
func (l *zapLogger) Info(args ...interface{}) {
	l.sugaredLogger.Info(args...)
}

// Warn logger
func (l *zapLogger) Warn(args ...interface{}) {
	l.sugaredLogger.Warn(args...)
}

// Error logger
func (l *zapLogger) Error(args ...interface{}) {
	l.sugaredLogger.Error(args...)
}

func (l *zapLogger) Fatal(args ...interface{}) {
	l.sugaredLogger.Fatal(args...)
}

func (l *zapLogger) Debugf(format string, args ...interface{}) {
	l.sugaredLogger.Debugf(format, args...)
}

func (l *zapLogger) Infof(format string, args ...interface{}) {
	l.sugaredLogger.Infof(format, args...)
}

func (l *zapLogger) Warnf(format string, args ...interface{}) {
	l.sugaredLogger.Warnf(format, args...)
}

func (l *zapLogger) Errorf(format string, args ...interface{}) {
	l.sugaredLogger.Errorf(format, args...)
}

func (l *zapLogger) Fatalf(format string, args ...interface{}) {
	l.sugaredLogger.Fatalf(format, args...)
}

func (l *zapLogger) Panicf(format string, args ...interface{}) {
	l.sugaredLogger.Panicf(format, args...)
}

func (l *zapLogger) WithFields(fields Fields) Logger {
	var f = make([]interface{}, 0)
	for k, v := range fields {
		f = append(f, k)
		f = append(f, v)
	}
	newLogger := l.sugaredLogger.With(f...)
	return &zapLogger{newLogger}
}
