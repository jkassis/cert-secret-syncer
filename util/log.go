package util

import (
	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func LogNew() logr.Logger {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // Enable colored level output

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}
	defer logger.Sync() // flushes buffer, if any
	return zapr.NewLogger(logger)
}
