package kitdriver

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/blendle/zapdriver"
	"go.uber.org/zap"
)

var (
	ErrKeyNotString       = errors.New("the key of each pair of keyvals should be string")
	ErrKeyValPairMismatch = errors.New("the keyvals should not match the condition (len(keyvals) < 2 || len(keyvals)%2 != 0)")
	ErrLogLevelNotFound   = errors.New("the first key should be one of the zap log levels (info, err, error, debug, warn, fatal, panic, dpanic)")
)

type GokitLogger interface {
	Log(keyvals ...interface{}) error
}

func NewProduction(options ...zap.Option) (*Logger, error) {
	zapLogger, err := zapdriver.NewProduction(options...)
	if err != nil {
		return nil, err
	}
	return &Logger{zapLogger}, nil
}

func NewDevelopment(options ...zap.Option) (*Logger, error) {
	zapLogger, err := zapdriver.NewDevelopment(options...)
	if err != nil {
		return nil, err
	}
	return &Logger{zapLogger}, nil
}

type Logger struct {
	zapLogger *zap.Logger
}

func (k *Logger) Log(keyvals ...interface{}) error {
	if len(keyvals) < 2 || len(keyvals)%2 != 0 {
		k.zapLogger.DPanic(ErrKeyValPairMismatch.Error())
		return ErrKeyValPairMismatch
	}

	fields := []zap.Field{
		zapdriver.SourceLocation(runtime.Caller(1)),
	}
	for i := 2; i < len(keyvals); i = i + 2 {
		key, err := k.key(keyvals[i])
		if err != nil {
			return err
		}
		value := fmt.Sprintf("%v", keyvals[i+1])
		fields = append(fields, zapdriver.Label(key, value))
	}

	level, err := k.key(keyvals[0])
	if err != nil {
		return err
	}
	msg := fmt.Sprintf("%v", keyvals[1])
	switch level {
	case "info":
		k.zapLogger.Info(msg, fields...)
	case "err", "error":
		k.zapLogger.Error(msg, fields...)
	case "debug":
		k.zapLogger.Debug(msg, fields...)
	case "warn":
		k.zapLogger.Warn(msg, fields...)
	case "fatal":
		k.zapLogger.Fatal(msg, fields...)
	case "panic":
		k.zapLogger.Panic(msg, fields...)
	case "dpanic":
		k.zapLogger.DPanic(msg, fields...)
	default:
		k.zapLogger.DPanic(ErrLogLevelNotFound.Error())
		return ErrLogLevelNotFound
	}

	return nil
}

func (k *Logger) key(key interface{}) (string, error) {
	s, ok := key.(string)
	if !ok {
		k.zapLogger.DPanic(ErrKeyNotString.Error())
		return "", ErrKeyNotString
	}
	return s, nil
}
