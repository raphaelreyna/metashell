package log

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var Log *Logger

func SetLog(root, component string) error {
	Log = &Logger{}
	return Log.init(root, component)
}

type Logger struct {
	out       *os.File
	component string
	dir       string

	zerolog.Logger
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

func (l *Logger) init(root, component string) error {
	l.dir = filepath.Join(root, component)

	if err := ensureDir(l.dir); err != nil {
		return err
	}

	if err := l.rotate(); err != nil {
		return err
	}

	l.component = component
	l.Logger = zerolog.New(l).
		With().
		Str("component", l.component).
		Logger()

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
