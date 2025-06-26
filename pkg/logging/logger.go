package logging

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gopkg.in/natefinch/lumberjack.v2"
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

	// File logging configuration
	EnableFile  bool   `yaml:"enable_file" json:"enable_file"`
	LogDir      string `yaml:"log_dir" json:"log_dir"`
	LogFileName string `yaml:"log_file_name" json:"log_file_name"`
	MaxSize     int    `yaml:"max_size" json:"max_size"`       // Max size in MB before rotation
	MaxBackups  int    `yaml:"max_backups" json:"max_backups"` // Max number of old files to keep
	MaxAge      int    `yaml:"max_age" json:"max_age"`         // Max days to keep old files
	Compress    bool   `yaml:"compress" json:"compress"`       // Compress old files
}

// DefaultConfig returns a default logging configuration
func DefaultConfig() Config {
	return Config{
		Level:      LevelInfo,
		Pretty:     true,
		TimeFormat: time.RFC3339,

		// File logging defaults
		EnableFile:  true,
		LogDir:      "logs",
		LogFileName: "backtester.log",
		MaxSize:     10,   // 10MB
		MaxBackups:  5,    // Keep 5 old files
		MaxAge:      30,   // Keep files for 30 days
		Compress:    true, // Compress old files
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

	var writers []io.Writer

	// Always add console output
	if config.Pretty {
		consoleWriter := zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		}
		writers = append(writers, consoleWriter)
	} else {
		writers = append(writers, os.Stderr)
	}

	// Add file output if enabled
	if config.EnableFile {
		// Create log directory if it doesn't exist
		if err := os.MkdirAll(config.LogDir, 0755); err != nil {
			// If we can't create the log directory, log to stderr and continue
			logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
			logger.Error().Err(err).Str("log_dir", config.LogDir).Msg("Failed to create log directory")
		} else {
			// Set up rolling file logger
			fileWriter := &lumberjack.Logger{
				Filename:   filepath.Join(config.LogDir, config.LogFileName),
				MaxSize:    config.MaxSize,
				MaxBackups: config.MaxBackups,
				MaxAge:     config.MaxAge,
				Compress:   config.Compress,
			}
			writers = append(writers, fileWriter)
		}
	}

	// Create multi-writer that writes to both console and file
	var output io.Writer
	if len(writers) == 1 {
		output = writers[0]
	} else {
		output = io.MultiWriter(writers...)
	}

	// Configure the global logger
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
}

// GetLogger returns a logger with the specified component name
func GetLogger(component string) zerolog.Logger {
	return log.With().Str("component", component).Logger()
}

// GetSubLogger returns a logger with additional context
func GetSubLogger(parent zerolog.Logger, subComponent string) zerolog.Logger {
	return parent.With().Str("subcomponent", subComponent).Logger()
}

// ConfigWithFileLogging creates a config with file logging enabled
func ConfigWithFileLogging(level LogLevel, pretty bool, logDir string, fileName string) Config {
	return Config{
		Level:      level,
		Pretty:     pretty,
		TimeFormat: time.RFC3339,

		// File logging configuration
		EnableFile:  true,
		LogDir:      logDir,
		LogFileName: fileName,
		MaxSize:     10,   // 10MB
		MaxBackups:  5,    // Keep 5 old files
		MaxAge:      30,   // Keep files for 30 days
		Compress:    true, // Compress old files
	}
}
