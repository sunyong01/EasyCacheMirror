package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

func init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	config.OutputPaths = []string{"stdout"}

	// 从环境变量获取日志级别，默认为 INFO
	logLevel := os.Getenv("LOG_LEVEL")
	switch logLevel {
	case "DEBUG":
		config.Level.SetLevel(zap.DebugLevel)
	case "WARN":
		config.Level.SetLevel(zap.WarnLevel)
	case "ERROR":
		config.Level.SetLevel(zap.ErrorLevel)
	default:
		config.Level.SetLevel(zap.InfoLevel)
	}

	var err error
	log, err = config.Build()
	if err != nil {
		panic(err)
	}
}

// GetLogger 返回全局logger实例
func GetLogger() *zap.Logger {
	return log
}
