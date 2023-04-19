package logger

import (
	"fmt"
	"sort"
	"strings"
)

// Level type
type Level uint32

const (
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	// Commonly used for hooks to send errors to an error tracking service.
	ErrorLevel Level = iota
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
	// TraceLevel level. Designates finer-grained informational events than the Debug.
	TraceLevel
)

var LevelMap = map[Level]string{
	ErrorLevel: "error",
	WarnLevel:  "warn",
	InfoLevel:  "info",
	DebugLevel: "debug",
	TraceLevel: "trace",
}

type LogPayload struct {
	Level   Level
	Fields  map[string]interface{}
	Error   error
	Message string
}

type LogFunc func(payload LogPayload)

func NoopLogFunc(payload LogPayload) {}

func NewNoopLogger() *LogWrapper {
	return NewLogWrapper(NoopLogFunc, map[string]interface{}{})
}

// NewSimpleLogFunc returns a simple logging func
func NewSimpleLogFunc(level Level) LogFunc {
	return func(payload LogPayload) {
		if level < payload.Level {
			return
		}

		fields := []string{}
		m := map[string]interface{}{}
		keys := []string{"msg", "level", "error"}

		for k, v := range payload.Fields {
			if k != "msg" && k != "level" {
				keys = append(keys, k)
				m[k] = v
			}
		}

		m["msg"] = payload.Message
		m["level"] = LevelMap[level]

		if payload.Error != nil {
			m["error"] = payload.Error
		}

		sort.Strings(keys)

		for _, k := range keys {
			v := m[k]
			fields = append(fields, fmt.Sprintf("%s=%q", k, v))
		}

		fmt.Println(strings.Join(fields, " "))
	}
}

type LogWrapper struct {
	LogFunc LogFunc
	Fields  map[string]interface{}
	Error   error
}

// NewLogWrapper returns a new log wrapper
func NewLogWrapper(logFunc LogFunc, fields map[string]interface{}) *LogWrapper {
	if fields == nil {
		fields = map[string]interface{}{}
	}

	return &LogWrapper{
		LogFunc: logFunc,
		Fields:  fields,
	}
}

// clone clones a log wrapper to iteratively build the log
func (l *LogWrapper) clone() *LogWrapper {
	newWrapper := &LogWrapper{
		LogFunc: l.LogFunc,
		Error:   l.Error,
		Fields:  map[string]interface{}{},
	}

	for k, v := range l.Fields {
		newWrapper.Fields[k] = v
	}

	return newWrapper
}

func (l *LogWrapper) WithError(err error) *LogWrapper {
	newWrapper := l.clone()
	newWrapper.Error = err
	return newWrapper
}

func (l *LogWrapper) WithField(key string, value interface{}) *LogWrapper {
	newWrapper := l.clone()
	newWrapper.Fields[key] = value
	return newWrapper
}

func (l *LogWrapper) Tracef(format string, v ...interface{}) {
	l.LogFunc(LogPayload{
		Level:   TraceLevel,
		Fields:  l.Fields,
		Error:   l.Error,
		Message: fmt.Sprintf(format, v...),
	})
}

func (l *LogWrapper) Debugf(format string, v ...interface{}) {
	l.LogFunc(LogPayload{
		Level:   DebugLevel,
		Fields:  l.Fields,
		Error:   l.Error,
		Message: fmt.Sprintf(format, v...),
	})
}

func (l *LogWrapper) Errorf(format string, v ...interface{}) {
	l.LogFunc(LogPayload{
		Level:   ErrorLevel,
		Fields:  l.Fields,
		Error:   l.Error,
		Message: fmt.Sprintf(format, v...),
	})
}

func (l *LogWrapper) Warnf(format string, v ...interface{}) {
	l.LogFunc(LogPayload{
		Level:   WarnLevel,
		Fields:  l.Fields,
		Error:   l.Error,
		Message: fmt.Sprintf(format, v...),
	})
}

func (l *LogWrapper) Infof(format string, v ...interface{}) {
	l.LogFunc(LogPayload{
		Level:   InfoLevel,
		Fields:  l.Fields,
		Error:   l.Error,
		Message: fmt.Sprintf(format, v...),
	})
}
