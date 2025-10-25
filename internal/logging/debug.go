package logging

import (
	"log/slog"
	"os"
	"strings"
)

// Logger provides topic-based debug logging with minimal overhead when disabled
type Logger struct {
	topic   string
	enabled bool
}

var enabledTopics = make(map[string]bool)

func init() {
	// Read DEBUG_TOPICS env var: DEBUG_TOPICS=atr,sma,strategy
	topics := os.Getenv("DEBUG_TOPICS")
	if topics == "" {
		return
	}

	// Special case: "all" enables everything
	if topics == "all" {
		enabledTopics["*"] = true
		configureSlog()
		return
	}

	// Parse comma-separated topics eg. DEBUG_TOPICS=atr,sma,strategy
	for _, topic := range strings.Split(topics, ",") {
		topic = strings.TrimSpace(topic)
		if topic != "" {
			enabledTopics[topic] = true
		}
	}

	// Configure slog to DEBUG level when any topics are enabled
	if len(enabledTopics) > 0 {
		configureSlog()
	}
}

// configureSlog sets slog's default logger to DEBUG level
func configureSlog() {
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	handler := slog.NewTextHandler(os.Stderr, opts)
	slog.SetDefault(slog.New(handler))
}

// New creates a new topic-specific logger
// Usage: var atrLog = logging.New("atr")
func New(topic string) *Logger {
	enabled := enabledTopics["*"] || enabledTopics[topic]
	return &Logger{
		topic:   topic,
		enabled: enabled,
	}
}

// Debug logs a debug message if this topic is enabled
// Fast path: returns immediately if disabled (single bool check)
func (l *Logger) Debug(msg string, args ...any) {
	if !l.enabled {
		return
	}
	// Prepend topic to args for context
	allArgs := append([]any{"topic", l.topic}, args...)
	slog.Debug(msg, allArgs...)
}

// Info logs an info message if this topic is enabled
func (l *Logger) Info(msg string, args ...any) {
	if !l.enabled {
		return
	}
	allArgs := append([]any{"topic", l.topic}, args...)
	slog.Info(msg, allArgs...)
}

// Warn logs a warning message if this topic is enabled
func (l *Logger) Warn(msg string, args ...any) {
	if !l.enabled {
		return
	}
	allArgs := append([]any{"topic", l.topic}, args...)
	slog.Warn(msg, allArgs...)
}

// Enabled returns true if this logger is enabled
// Useful for expensive computations: if log.Enabled() { ... }
func (l *Logger) Enabled() bool {
	return l.enabled
}
