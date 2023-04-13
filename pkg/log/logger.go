package log

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"

	nkruntime "github.com/heroiclabs/nakama-common/runtime"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var instance *Logger
var once sync.Once

type Logger struct {
	log nkruntime.Logger
}

type LoggingFormat int8

const (
	JSONFormat LoggingFormat = iota - 1
	StackdriverFormat
)

func StackdriverLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch l {
	case zapcore.DebugLevel:
		enc.AppendString("DEBUG")
	case zapcore.InfoLevel:
		enc.AppendString("INFO")
	case zapcore.WarnLevel:
		enc.AppendString("WARNING")
	case zapcore.ErrorLevel:
		enc.AppendString("ERROR")
	case zapcore.DPanicLevel:
		enc.AppendString("CRITICAL")
	case zapcore.PanicLevel:
		enc.AppendString("CRITICAL")
	case zapcore.FatalLevel:
		enc.AppendString("CRITICAL")
	default:
		enc.AppendString("DEFAULT")
	}
}

// Create a new JSON log encoder with the correct settings.
func newJSONEncoder(format LoggingFormat) zapcore.Encoder {
	if format == StackdriverFormat {
		return zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:        "time",
			LevelKey:       "severity",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			EncodeLevel:    StackdriverLevelEncoder,
			EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		})
	}

	return zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	})
}

func NewJSONLogger(output *os.File, level zapcore.Level, format LoggingFormat) *zap.Logger {
	jsonEncoder := newJSONEncoder(format)

	core := zapcore.NewCore(jsonEncoder, zapcore.Lock(output), level)
	options := []zap.Option{zap.AddCaller()}
	return zap.New(core, options...)
}

func loggerForTest() *zap.Logger {
	return NewJSONLogger(os.Stdout, zapcore.InfoLevel, JSONFormat)
}

func getInstance() *Logger {
	once.Do(func() {
		instance = &Logger{
			log: NewRuntimeGoLogger(loggerForTest()),
		}
	})
	return instance
}

func GetLogger() nkruntime.Logger {
	return getInstance().log
}

type RuntimeGoLogger struct {
	logger *zap.Logger
	fields map[string]interface{}
}

func NewRuntimeGoLogger(logger *zap.Logger) nkruntime.Logger {
	return &RuntimeGoLogger{
		fields: make(map[string]interface{}),
		logger: logger.WithOptions(zap.AddCallerSkip(1)).With(zap.String("runtime", "go")),
	}
}

func (l *RuntimeGoLogger) getFileLine() zap.Field {
	_, filename, line, ok := runtime.Caller(2)
	if !ok {
		return zap.Skip()
	}
	filenameSplit := strings.SplitN(filename, "@", 2)
	if len(filenameSplit) >= 2 {
		filename = filenameSplit[1]
	}
	return zap.String("source", fmt.Sprintf("%v:%v", filename, line))
}

func (l *RuntimeGoLogger) Debug(format string, v ...interface{}) {
	if l.logger.Core().Enabled(zap.DebugLevel) {
		msg := fmt.Sprintf(format, v...)
		l.logger.Debug(msg)
	}
}

func (l *RuntimeGoLogger) Info(format string, v ...interface{}) {
	if l.logger.Core().Enabled(zap.InfoLevel) {
		msg := fmt.Sprintf(format, v...)
		l.logger.Info(msg)
	}
}

func (l *RuntimeGoLogger) Warn(format string, v ...interface{}) {
	if l.logger.Core().Enabled(zap.WarnLevel) {
		msg := fmt.Sprintf(format, v...)
		l.logger.Warn(msg, l.getFileLine())
	}
}

func (l *RuntimeGoLogger) Error(format string, v ...interface{}) {
	if l.logger.Core().Enabled(zap.ErrorLevel) {
		msg := fmt.Sprintf(format, v...)
		l.logger.Error(msg, l.getFileLine())
	}
}

func (l *RuntimeGoLogger) WithField(key string, v interface{}) nkruntime.Logger {
	return l.WithFields(map[string]interface{}{key: v})
}

func (l *RuntimeGoLogger) WithFields(fields map[string]interface{}) nkruntime.Logger {
	f := make([]zap.Field, 0, len(fields)+len(l.fields))
	newFields := make(map[string]interface{}, len(fields)+len(l.fields))
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		if k == "runtime" {
			continue
		}
		newFields[k] = v
		f = append(f, zap.Any(k, v))
	}

	return &RuntimeGoLogger{
		logger: l.logger.With(f...),
		fields: newFields,
	}
}

func (l *RuntimeGoLogger) Fields() map[string]interface{} {
	return l.fields
}
