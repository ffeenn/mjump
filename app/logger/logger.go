package logger

import (
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

// var logLevels = map[string]logrus.Level{
// 	"DEBUG": logrus.DebugLevel,
// 	"INFO":  logrus.InfoLevel,
// 	"WARN":  logrus.WarnLevel,
// 	"ERROR": logrus.ErrorLevel,
// }

const (
	// Default log format will output [INFO]: 2006-01-02T15:04:05Z07:00 - Log message
	defaultLogFormat       = "[%lvl%]: %time% - %msg%"
	defaultTimestampFormat = time.RFC3339
)

func Initlog() {

	formatter := &Formatter{
		LogFormat:       "%time% [%lvl%] %msg%",
		TimestampFormat: "2006-01-02 15:04:05",
	}
	logger.SetFormatter(formatter)
	logger.SetOutput(os.Stdout)
	logger.SetLevel(logrus.DebugLevel)

}
func Debug(args ...interface{}) {
	logger.Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	logger.Debugf(format, args...)
}

func Info(args ...interface{}) {
	logger.Info(args...)
}

func Infof(format string, args ...interface{}) {
	logger.Infof(format, args...)
}

func Warn(args ...interface{}) {
	logger.Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	logger.Warnf(format, args...)
}

func Error(args ...interface{}) {
	logger.Error(args...)
}

func Errorf(format string, args ...interface{}) {
	logger.Errorf(format, args...)
}

func Panic(args ...interface{}) {
	logrus.Panic(args...)
}

func Fatal(args ...interface{}) {
	logrus.Fatal(args...)
}

type Formatter struct {
	// Timestamp format
	TimestampFormat string
	// Available standard keys: time, msg, lvl
	// Also can include custom fields but limited to strings.
	// All of fields need to be wrapped inside %% i.e %time% %msg%
	LogFormat string

	// Disables the truncation of the level text to 4 characters.
	DisableLevelTruncation bool
}

// Format building log message.
func (f *Formatter) Format(entry *logrus.Entry) ([]byte, error) {
	output := f.LogFormat
	if output == "" {
		output = defaultLogFormat
	}

	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = defaultTimestampFormat
	}

	output = strings.Replace(output, "%time%", entry.Time.Format(timestampFormat), 1)
	output = strings.Replace(output, "%msg%", entry.Message, 1)
	level := strings.ToUpper(entry.Level.String())
	if !f.DisableLevelTruncation {
		level = level[:4]
	}
	output = strings.Replace(output, "%lvl%", level, 1)

	for k, v := range entry.Data {
		if s, ok := v.(string); ok {
			output = strings.Replace(output, "%"+k+"%", s, 1)
		}
	}
	output += "\n"

	return []byte(output), nil
}
