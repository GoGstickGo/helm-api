package helmutils

import (
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

// Setup logger
func setupLogger(logger Logger) Logger {
	if logger != nil {
		return logger
	}

	defaultLogger := logrus.New()
	defaultLogger.SetLevel(logrus.InfoLevel)
	defaultLogger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})

	// Start log level monitoring
	go monitorLogLevel(defaultLogger)

	return defaultLogger
}

func monitorLogLevel(logger *logrus.Logger) {
	for {
		if level, err := logrus.ParseLevel(os.Getenv("HELM_API_LOG_LEVEL")); err == nil {
			logger.SetLevel(level)
		}
		time.Sleep(time.Second * 30)
	}
}
