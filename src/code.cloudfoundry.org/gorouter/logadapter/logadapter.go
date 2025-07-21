package logadapter

import (
	grlog "code.cloudfoundry.org/gorouter/logger"
	"code.cloudfoundry.org/lager/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log/slog"
	"os"
	"strings"
)

// zapLevelSink is a lager sink that uses a slog.Logger for logging.
// It implements the lager.Sink interface, allowing it to be used with
// lager's logging system. The Log method logs the message and source
// using the slog.Logger, and the LogLevel method returns the current
// logging level.
type zapLevelSink struct {
	logger *slog.Logger
}

func NewZapLevelSink(l *slog.Logger) *zapLevelSink {
	return &zapLevelSink{logger: l}
}

func (z *zapLevelSink) LogLevel() lager.LogLevel {
	switch strings.ToLower(grlog.GetLoggingLevel()) {
	case "debug":
		return lager.DEBUG
	case "info":
		return lager.INFO
	case "error":
		return lager.ERROR
	case "warn":
		// lager does not have a separate WARN level, it uses INFO for warnings.
		return lager.INFO
	case "fatal":
		return lager.FATAL
	default:
		return lager.INFO
	}
}

// The SetMinLevel method updates the logging level of the
// zapLevelSink based on the provided lager.LogLevel, mapping it to
// the corresponding zapcore.Level. The logger is expected to be set
// before using this sink, and it will log an info message when the
// logging level is updated.
func (z *zapLevelSink) SetMinLevel(level lager.LogLevel) {
	var zapLevel zapcore.Level
	switch level {
	case lager.DEBUG:
		zapLevel = zapcore.DebugLevel
	case lager.INFO:
		zapLevel = zapcore.InfoLevel
	case lager.ERROR:
		zapLevel = zapcore.ErrorLevel
	case lager.FATAL:
		zapLevel = zapcore.FatalLevel
	default:
		// There is no lager.WARN level, so mapping it to zapcore.WarnLevel in default case.
		zapLevel = zapcore.WarnLevel
	}

	grlog.SetLoggingLevel(zapLevel.String())

	// Print the new log level to the logger to confirm the change.
	// This is useful for debugging and confirming that the log level has been updated.
	if z.logger != nil {
		// We cannot use z.logger.Info() directly here because it won't print the
		// log level when it is set to zapcore.ErrorLevel or zapcore.FatalLevel.
		// Instead, we use slog.New() to log the message. This ensures that the
		// log level change is always logged, regardless of the current log level.
		tmpLogger := NewZapLoggerWithTimestamp()
		tmpLogger.Info("Gorouter logger -> zapcore log level updated.",
			zap.String("new log_level", zapLevel.String()))
	}
}

// NewZapLoggerWithTimestamp creates a new zap.Logger with a custom
// timestamp encoder that outputs the timestamp as a float64 representing
// the number of seconds since the Unix epoch.
func NewZapLoggerWithTimestamp() *zap.Logger {
	cfg := zap.NewProductionEncoderConfig()
	// customized keys for the JSON encoder
	cfg.TimeKey = "timestamp"
	cfg.LevelKey = "log_level"
	cfg.MessageKey = "message"
	// time encoded in RFC3339 format
	cfg.EncodeTime = zapcore.RFC3339TimeEncoder
	// Encode log level as int. zapcore.Level is 0-indexed,
	// so add 1 to the level to match the gorouter's expected level.
	cfg.EncodeLevel = func(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendInt(int(l + 1))
	}
	core := zapcore.NewCore(zapcore.NewJSONEncoder(cfg), zapcore.AddSync(os.Stdout), zap.InfoLevel)
	return zap.New(core)
}
