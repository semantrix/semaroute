package observability

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LoggerConfig holds configuration for the logger.
type LoggerConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"` // json or console
	OutputPath string `mapstructure:"output_path"`
	ErrorPath  string `mapstructure:"error_path"`
	Development bool   `mapstructure:"development"`
}

// NewLogger creates a new configured logger instance.
func NewLogger(config LoggerConfig) (*zap.Logger, error) {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(config.Level)); err != nil {
		level = zapcore.InfoLevel
	}

	// Create encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	// Choose encoder based on format
	var encoder zapcore.Encoder
	if config.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create core
	var core zapcore.Core

	if config.Development {
		// Development mode: log to console with more verbose output
		core = zapcore.NewCore(
			encoder,
			zapcore.AddSync(os.Stdout),
			level,
		)
	} else {
		// Production mode: log to file
		outputFile, err := os.OpenFile(config.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}

		errorFile, err := os.OpenFile(config.ErrorPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			outputFile.Close()
			return nil, err
		}

		// Create a tee core that writes to both files
		core = zapcore.NewTee(
			zapcore.NewCore(
				encoder,
				zapcore.AddSync(outputFile),
				level,
			),
			zapcore.NewCore(
				encoder,
				zapcore.AddSync(errorFile),
				zapcore.ErrorLevel,
			),
		)
	}

	// Create logger with options
	options := []zap.Option{
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	}

	if config.Development {
		options = append(options, zap.Development())
	}

	logger := zap.New(core, options...)

	return logger, nil
}

// DefaultLogger creates a logger with sensible defaults.
func DefaultLogger() *zap.Logger {
	logger, err := NewLogger(LoggerConfig{
		Level:      "info",
		Format:     "json",
		OutputPath: "logs/app.log",
		ErrorPath:  "logs/error.log",
		Development: false,
	})

	if err != nil {
		// Fallback to a basic logger if configuration fails
		config := zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		
		logger, _ = config.Build()
	}

	return logger
}

// SyncLogger ensures all buffered logs are written before shutdown.
func SyncLogger(logger *zap.Logger) {
	_ = logger.Sync()
}
