package logger

import "fmt"

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

type Logger interface {
	Errorf(format string, data ...interface{})
	Warnf(format string, data ...interface{})
	Infof(format string, data ...interface{})
	Debugf(format string, data ...interface{})
	Tracef(format string, data ...interface{})
}

type NoopLogger struct{}

func (n *NoopLogger) Errorf(format string, data ...interface{}) {}
func (n *NoopLogger) Warnf(format string, data ...interface{})  {}
func (n *NoopLogger) Infof(format string, data ...interface{})  {}
func (n *NoopLogger) Debugf(format string, data ...interface{}) {}
func (n *NoopLogger) Tracef(format string, data ...interface{}) {}

type SimpleLogger struct {
	level Level
}

func NewSimpleLogger() *SimpleLogger {
	return &SimpleLogger{
		level: InfoLevel,
	}
}

func (s *SimpleLogger) SetLevel(level Level) {
	s.level = level
}

func (s *SimpleLogger) logf(level, format string, data ...interface{}) {
	txt := fmt.Sprintf(format, data...)
	fmt.Printf("[%s] %s\n", level, txt)
}

func (s *SimpleLogger) Errorf(format string, data ...interface{}) {
	if s.level >= ErrorLevel {
		s.logf("ERROR", format, data...)
	}
}

func (s *SimpleLogger) Warnf(format string, data ...interface{}) {
	if s.level >= WarnLevel {
		s.logf("WARN", format, data...)
	}
}

func (s *SimpleLogger) Infof(format string, data ...interface{}) {
	if s.level >= InfoLevel {
		s.logf("INFO", format, data...)
	}
}

func (s *SimpleLogger) Debugf(format string, data ...interface{}) {
	if s.level >= DebugLevel {
		s.logf("DEBUG", format, data...)
	}
}

func (s *SimpleLogger) Tracef(format string, data ...interface{}) {
	if s.level >= TraceLevel {
		s.logf("TRACE", format, data...)
	}
}
