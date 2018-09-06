package kitdriver

import (
	"errors"
	"fmt"
	"runtime"
	"runtime/debug"

	"github.com/blendle/zapdriver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	INFO   = "info"
	ERR    = "err"
	ERROR  = "error"
	DEBUG  = "debug"
	WARN   = "warn"
	FATAL  = "fatal"
	PANIC  = "panic"
	DPANIC = "dpanic"
)

var (
	ErrKeyNotString       = errors.New("the key of each pair of keyvals should be string")
	ErrKeyValPairMismatch = errors.New("the keyvals should not match the condition (len(keyvals) < 2 || len(keyvals)%2 != 0)")
	ErrLogLevelNotFound   = errors.New("the first key should be one of the zap log levels (info, err, error, debug, warn, fatal, panic, dpanic)")
)

type GokitLogger interface {
	Log(keyvals ...interface{}) error
}

type ServiceContext struct {
	Service string
	Version string
}

func (l *ServiceContext) MarshalLogObject(e zapcore.ObjectEncoder) error {
	e.AddString("service", l.Service)
	e.AddString("version", l.Version)
	return nil
}

func NewProduction(service, version string, options ...zap.Option) (*Logger, error) {
	config := alterConfig(zapdriver.NewProductionConfig())
	zapLogger, err := config.Build(append(options, zapdriver.WrapCore())...)
	if err != nil {
		return nil, err
	}
	return &Logger{zapLogger}, nil
}

func NewDevelopment(service, version string, options ...zap.Option) (*Logger, error) {
	config := alterConfig(zapdriver.NewDevelopmentConfig())
	zapLogger, err := config.Build(append(options, zapdriver.WrapCore())...)
	if err != nil {
		return nil, err
	}
	zapLogger = zapLogger.With(zap.Object("serviceContext", &ServiceContext{
		Service: service,
		Version: version,
	}))
	return &Logger{zapLogger}, nil
}

func alterConfig(config zap.Config) zap.Config {
	config.DisableStacktrace = true
	config.DisableCaller = true
	config.EncoderConfig.TimeKey = "timestamp"
	return config
}

type Logger struct {
	zapLogger *zap.Logger
}

func (k *Logger) Log(keyvals ...interface{}) error {
	if len(keyvals) < 2 || len(keyvals)%2 != 0 {
		k.zapLogger.DPanic(ErrKeyValPairMismatch.Error())
		return ErrKeyValPairMismatch
	}

	var fields []zap.Field
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
	case INFO:
		k.zapLogger.Info(msg, fields...)
	case ERR, ERROR:
		stack := string(debug.Stack())
		k.zapLogger.Error(msg+"\n"+stack, fields...)
	case DEBUG:
		fields = append(fields, zapdriver.SourceLocation(runtime.Caller(1)))
		k.zapLogger.Debug(msg, fields...)
	case WARN:
		k.zapLogger.Warn(msg, fields...)
	case FATAL:
		stack := string(debug.Stack())
		k.zapLogger.Fatal(msg+"\n"+stack, fields...)
	case PANIC:
		k.zapLogger.Panic(msg, fields...)
	case DPANIC:
		k.zapLogger.DPanic(msg, fields...)
	default:
		k.zapLogger.DPanic(ErrLogLevelNotFound.Error())
		return ErrLogLevelNotFound
	}

	return nil
}

func (k *Logger) Sync() {
	k.zapLogger.Sync()
}

func (k *Logger) key(key interface{}) (string, error) {
	s, ok := key.(string)
	if !ok {
		k.zapLogger.DPanic(ErrKeyNotString.Error())
		return "", ErrKeyNotString
	}
	return s, nil
}
