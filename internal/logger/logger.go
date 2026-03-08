package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Logger represents a custom logger instance
type Logger struct {
	*logrus.Logger
}

// Fields aliases log field maps so callers don't need to import logrus directly.
type Fields = logrus.Fields

// New creates a new Logger instance
func New(level, format string) *Logger {
	log := logrus.New()

	switch level {
	case "debug":
		log.SetLevel(logrus.DebugLevel)
	case "info":
		log.SetLevel(logrus.InfoLevel)
	case "warn":
		log.SetLevel(logrus.WarnLevel)
	case "error":
		log.SetLevel(logrus.ErrorLevel)
	default:
		log.SetLevel(logrus.InfoLevel)
	}

	if format == "json" {
		log.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	log.SetOutput(os.Stdout)

	return &Logger{Logger: log}
}

// WithField adds a new field to the logger
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields Fields) *logrus.Entry {
	return l.Logger.WithFields(fields)
}
