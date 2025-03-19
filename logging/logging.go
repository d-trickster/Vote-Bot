package logging

import (
	"log/slog"
	"os"
)

const (
	envDev  = "dev"
	envProd = "prod"
)

func SetupLogging(env, logPath string) *os.File {
	var logger *slog.Logger
	var logFile *os.File

	switch env {
	case envDev:
		logger = slog.New(NewTerminalHandler(os.Stdout, slog.LevelDebug))
	case envProd:
		var err error
		logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			panic("failed to open log file")
		}
		logger = slog.New(slog.NewJSONHandler(logFile, nil))
	}

	if logger == nil {
		panic("Logging setup has falied")
	}
	slog.SetDefault(logger)

	return logFile
}
