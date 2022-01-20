package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
	"sync"
)

var testLoggerMapping = make(map[string]*testLogger, 0)
var initLock sync.Mutex

type testLogger struct {
	tag          string
	fileLogger   *zap.SugaredLogger
	stdoutLogger *zap.SugaredLogger
}

func (l *testLogger) Write(p []byte) (int, error) {
	l.stdoutLogger.Info(string(p))
	l.fileLogger.Info(string(p))
	return len(p), nil
}

func (l *testLogger) Info(args ...interface{}) {
	l.stdoutLogger.Info(args...)
	l.fileLogger.Info(args...)
}

func (l *testLogger) Infof(template string, args ...interface{}) {
	l.stdoutLogger.Infof(template, args...)
	l.fileLogger.Infof(template, args...)
}

func AllTestLogsLocations() []string {
	result := make([]string, 1)
	initLock.Lock()
	defer initLock.Unlock()
	for _, logger := range testLoggerMapping {
		if logger != nil {
			result = append(result, dirForTestCaseLog(logger.tag))
		}
	}
	return result
}

func TestLogger(tag string) *testLogger {
	initLock.Lock()
	defer initLock.Unlock()

	if logger, ok := testLoggerMapping[tag]; !ok {
		if logger, ok := testLoggerMapping[tag]; ok {
			return logger
		}

		rolling := &lumberjack.Logger{
			Filename:   dirForTestCaseLog(tag),
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
		fileLogsConfig := zapcore.NewCore(encoder, writeSyncer, zapcore.DebugLevel)

		cfg := zap.NewProductionEncoderConfig()
		cfg.EncodeTime = CustomTimeEncoder
		cfg.EncodeLevel = CustomLevelEncoder

		unFormatEncoder := zapcore.NewConsoleEncoder(cfg)
		unFormatStdoutConfig := zapcore.NewCore(unFormatEncoder, zapcore.AddSync(os.Stdout), zapcore.DebugLevel)

		logger := testLogger{
			tag:          tag,
			fileLogger:   zap.New(fileLogsConfig).Sugar(),
			stdoutLogger: zap.New(unFormatStdoutConfig).Sugar(),
		}

		testLoggerMapping[tag] = &logger
		return &logger
	} else {
		return logger
	}
}

func dirForTestCaseLog(tag string) string {
	return filepath.Join("testlog", tag)
}
