package logging

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LogLevel represents the logging level
type LogLevel string

const (
	LevelTrace LogLevel = "trace"
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
	LevelFatal LogLevel = "fatal"
	LevelPanic LogLevel = "panic"
)

// Config holds logging configuration
type Config struct {
	Level      LogLevel `yaml:"level" json:"level"`
	Pretty     bool     `yaml:"pretty" json:"pretty"`
	TimeFormat string   `yaml:"time_format" json:"time_format"`
}

// DefaultConfig returns a default logging configuration
func DefaultConfig() Config {
	return Config{
		Level:      LevelInfo,
		Pretty:     true,
		TimeFormat: time.RFC3339,
	}
}

// Initialize sets up the global logger with the given configuration
func Initialize(config Config) {
	// Set global log level
	switch config.Level {
	case LevelTrace:
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case LevelDebug:
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case LevelInfo:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case LevelWarn:
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case LevelError:
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case LevelFatal:
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case LevelPanic:
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	// Configure time format
	zerolog.TimeFieldFormat = config.TimeFormat

	// Configure output format
	if config.Pretty {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		})
	} else {
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}
}

// GetLogger returns a logger with the specified component name
func GetLogger(component string) zerolog.Logger {
	return log.With().Str("component", component).Logger()
}

// GetSubLogger returns a logger with additional context
func GetSubLogger(parent zerolog.Logger, subComponent string) zerolog.Logger {
	return parent.With().Str("subcomponent", subComponent).Logger()
}
