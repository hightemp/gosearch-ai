package log

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

func New() zerolog.Logger {
	zerolog.TimeFieldFormat = time.RFC3339Nano
	level := zerolog.InfoLevel
	if os.Getenv("APP_ENV") == "dev" {
		level = zerolog.DebugLevel
	}
	logger := zerolog.New(os.Stdout).Level(level).With().Timestamp().Logger()
	return logger
}
