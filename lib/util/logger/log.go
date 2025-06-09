package logger

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	logger := zerolog.New(os.Stderr).With().Timestamp().Caller().Logger()

	log.Logger = logger
}

// Logger provides structured logging with scope management
type Logger struct {
	logger    zerolog.Logger
	scope     string
	startTime time.Time
}

// NewLogger creates a new logger instance with a specific scope
func NewLogger(scope string) *Logger {
	return &Logger{
		logger:    log.With().Str("scope", scope).Logger(),
		scope:     scope,
		startTime: time.Now(),
	}
}

// NewLoggerWithContext creates a new logger instance with context
func NewLoggerWithContext(ctx context.Context, scope string) *Logger {
	logger := log.With().Str("scope", scope)

	// Add context values if available
	if requestID := ctx.Value("request_id"); requestID != nil {
		logger = logger.Str("request_id", requestID.(string))
	}

	return &Logger{
		logger:    logger.Logger(),
		scope:     scope,
		startTime: time.Now(),
	}
}

// Start logs the beginning of a scope
func (l *Logger) Start() *Logger {
	l.logger.Info().
		Str("event", "scope_start").
		Time("start_time", l.startTime).
		Msg("Starting scope")
	return l
}

// StartWithMsg logs the beginning of a scope with a custom message
func (l *Logger) StartWithMsg(msg string) *Logger {
	l.logger.Info().
		Str("event", "scope_start").
		Time("start_time", l.startTime).
		Msg(msg)
	return l
}

// End logs the completion of a scope with execution time
func (l *Logger) End() {
	duration := time.Since(l.startTime)
	l.logger.Info().
		Str("event", "scope_end").
		Dur("duration", duration).
		Time("end_time", time.Now()).
		Msg("Scope completed")
}

// EndWithMsg logs the completion of a scope with execution time and custom message
func (l *Logger) EndWithMsg(msg string) {
	duration := time.Since(l.startTime)
	l.logger.Info().
		Str("event", "scope_end").
		Dur("duration", duration).
		Time("end_time", time.Now()).
		Msg(msg)
}

// EndWithError logs the completion of a scope with an error
func (l *Logger) EndWithError(err error) {
	duration := time.Since(l.startTime)
	l.logger.Error().
		Err(err).
		Str("event", "scope_error").
		Dur("duration", duration).
		Time("end_time", time.Now()).
		Msg("Scope failed")
}

// Info logs an info message
func (l *Logger) Info() *zerolog.Event {
	return l.logger.Info()
}

// Error logs an error message
func (l *Logger) Error() *zerolog.Event {
	return l.logger.Error()
}

// Debug logs a debug message
func (l *Logger) Debug() *zerolog.Event {
	return l.logger.Debug()
}

// Warn logs a warning message
func (l *Logger) Warn() *zerolog.Event {
	return l.logger.Warn()
}

// DataCreated logs data creation
func (l *Logger) DataCreated(entityType string, entityID string, additionalFields ...map[string]interface{}) {
	event := l.logger.Info().
		Str("event", "data_created").
		Str("entity_type", entityType).
		Str("entity_id", entityID)

	for _, fields := range additionalFields {
		for k, v := range fields {
			event = event.Interface(k, v)
		}
	}

	event.Msg("Data created")
}

// DataUpdated logs data updates
func (l *Logger) DataUpdated(entityType string, entityID string, changes map[string]interface{}, additionalFields ...map[string]interface{}) {
	event := l.logger.Info().
		Str("event", "data_updated").
		Str("entity_type", entityType).
		Str("entity_id", entityID).
		Interface("changes", changes)

	for _, fields := range additionalFields {
		for k, v := range fields {
			event = event.Interface(k, v)
		}
	}

	event.Msg("Data updated")
}

// DataDeleted logs data deletion
func (l *Logger) DataDeleted(entityType string, entityID string, additionalFields ...map[string]interface{}) {
	event := l.logger.Info().
		Str("event", "data_deleted").
		Str("entity_type", entityType).
		Str("entity_id", entityID)

	for _, fields := range additionalFields {
		for k, v := range fields {
			event = event.Interface(k, v)
		}
	}

	event.Msg("Data deleted")
}

// WithField adds a field to the logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		logger:    l.logger.With().Interface(key, value).Logger(),
		scope:     l.scope,
		startTime: l.startTime,
	}
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newLogger := l.logger.With()
	for k, v := range fields {
		newLogger = newLogger.Interface(k, v)
	}
	return &Logger{
		logger:    newLogger.Logger(),
		scope:     l.scope,
		startTime: l.startTime,
	}
}
