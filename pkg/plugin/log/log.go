package log

import (
	"os"

	"github.com/hashicorp/go-hclog"
	"github.com/raphaelreyna/metashell/pkg/plugin/proto/proto"
)

var logger hclog.Logger

func init() {
	logger = hclog.New(&hclog.LoggerOptions{
		Name:       "metashell-preinitialized-plugin-" + os.Args[0],
		Level:      hclog.LevelFromString("DEBUG"),
		JSONFormat: true,
	})
}

func GetLogger() hclog.Logger {
	return logger
}

// Init initializes the logger with the provided plugin configuration.
// It sets the log name and level based on the configuration.
// This function should be called at the start of the plugins Init method.
func Init(config *proto.PluginConfig) {
	logger = logger.ResetNamed(config.LogName)
	logger.SetLevel(hclog.LevelFromString(config.LogLevel))
}

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
