package zap

import (
	"fmt"
	. "ninego/log"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var _ LoggerInterface = (*ZapSugaredLogger)(nil)

type ZapSugaredLogger struct {
	logger    *zap.SugaredLogger
	zapConfig *zap.Config
}

func NewZapSugaredLogger() LoggerInterface {
	return buildZapLog()
}

func buildZapLog() LoggerInterface {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	zapConfig := &zap.Config{
		Level:             zap.NewAtomicLevelAt(zapcore.InfoLevel),
		Development:       true,
		DisableCaller:     false,
		DisableStacktrace: true,
		Sampling:          &zap.SamplingConfig{Initial: 100, Thereafter: 100},
		Encoding:          "json",
		EncoderConfig:     encoderConfig,
		OutputPaths:       []string{"stderr"},
		ErrorOutputPaths:  []string{"stderr"},
	}
	l, err := zapConfig.Build(zap.AddCallerSkip(2))
	if err != nil {
		fmt.Printf("zap build logger fail err=%v", err)
		return nil
	}
	return &ZapSugaredLogger{
		logger:    l.Sugar(),
		zapConfig: zapConfig,
	}
}

func levelToZapLevel(level Level) zapcore.Level {
	levelMap := map[Level]zapcore.Level{
		LevelDebug: zapcore.DebugLevel,
		LevelInfo:  zapcore.InfoLevel,
		LevelWarn:  zapcore.WarnLevel,
		LevelError: zapcore.ErrorLevel,
		LevelPanic: zapcore.PanicLevel,
		LevelFatal: zapcore.FatalLevel,
	}
	if zapLevel, ok := levelMap[level]; ok {
		return zapLevel
	}
	return zapcore.ErrorLevel
}

func (l *ZapSugaredLogger) SetLevel(level Level) {
	l.zapConfig.Level.SetLevel(levelToZapLevel(level))
}

func (l *ZapSugaredLogger) Debug(msg string, v ...interface{}) {
	l.logger.Debugw(msg, ArgsToKeyValues(v...)...)
}

func (l *ZapSugaredLogger) Warn(msg string, v ...interface{}) {
	l.logger.Warnw(msg, ArgsToKeyValues(v...)...)
}

func (l *ZapSugaredLogger) Error(msg string, v ...interface{}) {
	l.logger.Errorw(msg, ArgsToKeyValues(v...)...)
}

func (l *ZapSugaredLogger) Panic(msg string, v ...interface{}) {
	l.logger.Panicw(msg, ArgsToKeyValues(v...)...)
}

func (l *ZapSugaredLogger) Fatal(msg string, v ...interface{}) {
	l.logger.Fatalw(msg, ArgsToKeyValues(v...)...)
}

func (l *ZapSugaredLogger) Info(msg string, v ...interface{}) {
	l.logger.Infow(msg, ArgsToKeyValues(v...)...)
}

// Close 关闭日志，释放资源
func (l *ZapSugaredLogger) Close() error {
	return l.logger.Sync()
}
