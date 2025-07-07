package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
)

var logger *Logger

func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}

func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

func Error(msg string, err error, args ...any) {
	logger.Error(msg, append(args, "error", err)...)
}

func Named(name string) hclog.Logger {
	return logger.Named(name)
}

func With(args ...any) hclog.Logger {
	if len(args)%2 != 0 {
		panic("With() requires an even number of arguments")
	}

	return logger.With(args...)
}

func GetLogLevel() string {
	return logger.GetLevel().String()
}

func SetLog(level, root, component string) error {
	logger = &Logger{}
	return logger.init(level, filepath.Join(root, "logs"), component)
}

func GetLogger() hclog.Logger {
	return logger
}

type Logger struct {
	out       *os.File
	component string
	dir       string

	hclog.Logger
	sync.Mutex
}

func (l *Logger) OutFilePath() string {
	return l.out.Name()
}

func (l *Logger) rotate() error {
	l.Lock()
	defer l.Unlock()

	var err error
	if l.out != nil {
		if err = l.out.Close(); err != nil {
			return err
		}
	}

	newLogFileName := fmt.Sprintf("%d.log", time.Now().Unix())
	l.out, err = os.Create(filepath.Join(l.dir, newLogFileName))
	if err != nil {
		err = fmt.Errorf("error rotating log file: %w", err)
	}
	return err
}

func (l *Logger) init(level, root, component string) error {
	if level == "" {
		level = "INFO"
	}
	if level != "DEBUG" && level != "INFO" && level != "WARN" && level != "ERROR" {
		return fmt.Errorf("invalid log level: %s", level)
	}

	l.dir = filepath.Join(root, component)

	if err := ensureDir(l.dir); err != nil {
		return err
	}

	if err := l.rotate(); err != nil {
		return err
	}

	l.component = component
	l.Logger = hclog.New(&hclog.LoggerOptions{
		Name:       component,
		Level:      hclog.LevelFromString(level),
		JSONFormat: true,
		Output:     l.out,
	})

	return nil
}

func (l *Logger) Write(p []byte) (int, error) {
	l.Lock()
	defer l.Unlock()
	return l.out.Write(p)
}

func ensureDir(path string) error {
	_, err := os.Stat(path)

	switch {
	case err == nil:
		return nil
	case !os.IsNotExist(err):
		return err
	}

	err = os.MkdirAll(path, 0700)
	if os.IsExist(err) {
		err = nil
	}

	return err
}
