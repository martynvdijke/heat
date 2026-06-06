package logger

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Levels
const (
	LevelDebug = "DEBUG"
	LevelInfo  = "INFO"
	LevelWarn  = "WARN"
	LevelError = "ERROR"
)

var levelOrder = map[string]int{
	LevelDebug: 0,
	LevelInfo:  1,
	LevelWarn:  2,
	LevelError: 3,
}

// LogEntry represents a structured log entry before serialization
type LogEntry struct {
	Timestamp time.Time
	Level     string
	Module    string
	Message   string
	Data      map[string]interface{}
	TraceID   string
	SpanID    string
}

// Logger provides leveled structured logging with SQLite persistence and OTel bridging
type Logger struct {
	db     *sql.DB
	mu     sync.RWMutex
	levels map[string]string // module -> level
	tracer trace.Tracer
	logCh  chan LogEntry
	done   chan struct{}
	once   sync.Once
}

// New creates a new Logger
func New(db *sql.DB) *Logger {
	l := &Logger{
		db:     db,
		levels: make(map[string]string),
		tracer: otel.Tracer("heat-logger"),
		logCh:  make(chan LogEntry, 1000),
		done:   make(chan struct{}),
	}
	l.loadSettings()
	go l.processLoop()
	return l
}

// loadSettings reads log verbosity settings from the database
func (l *Logger) loadSettings() {
	l.mu.Lock()
	defer l.mu.Unlock()

	rows, err := l.db.Query("SELECT module, level FROM log_settings")
	if err != nil {
		return
	}
	defer rows.Close()

	clear := make(map[string]string)
	for rows.Next() {
		var module, level string
		if err := rows.Scan(&module, &level); err != nil {
			continue
		}
		clear[module] = level
	}
	if len(clear) == 0 {
		clear["default"] = LevelWarn
	}
	l.levels = clear
}

// RefreshSettings reloads log verbosity from the database (call after settings change)
func (l *Logger) RefreshSettings() {
	l.loadSettings()
}

// shouldLog checks if a message at the given level should be logged for the module
func (l *Logger) shouldLog(module, level string) bool {
	l.mu.RLock()
	configured, ok := l.levels[module]
	l.mu.RUnlock()

	if !ok {
		l.mu.RLock()
		configured = l.levels["default"]
		l.mu.RUnlock()
	}
	if configured == "" {
		configured = LevelWarn
	}
	return levelOrder[level] >= levelOrder[configured]
}

// log is the internal logging method
func (l *Logger) log(level, module, msg string, data map[string]interface{}) {
	if !l.shouldLog(module, level) {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Module:    module,
		Message:   msg,
		Data:      data,
	}

	// Capture trace context from the OTel span
	span := trace.SpanFromContext(context.Background())
	if span != nil && span.SpanContext().IsValid() {
		entry.TraceID = span.SpanContext().TraceID().String()
		entry.SpanID = span.SpanContext().SpanID().String()
	}

	// Also emit via OTel span event if there's an active span
	if span != nil && span.SpanContext().IsValid() {
		attrs := []attribute.KeyValue{
			attribute.String("log.level", level),
			attribute.String("log.module", module),
			attribute.String("log.message", msg),
		}
		if data != nil {
			if d, err := json.Marshal(data); err == nil {
				attrs = append(attrs, attribute.String("log.data", string(d)))
			}
		}
		span.AddEvent("log."+level, trace.WithAttributes(attrs...))
		if level == LevelError {
			span.SetStatus(codes.Error, msg)
		}
	}

	// Enqueue for async processing
	select {
	case l.logCh <- entry:
	default:
		log.Printf("[LOGGER] dropping log entry: %s [%s] %s", level, module, msg)
	}
}

// processLoop drains the log channel and writes to DB and OTel
func (l *Logger) processLoop() {
	for entry := range l.logCh {
		l.writeSQLite(entry)
		l.writeOTel(entry)
	}
	close(l.done)
}

// writeSQLite persists a log entry to the app_logs table with auto-prune
func (l *Logger) writeSQLite(entry LogEntry) {
	if l.db == nil {
		return
	}

	var dataJSON *string
	if entry.Data != nil {
		if d, err := json.Marshal(entry.Data); err == nil {
			s := string(d)
			dataJSON = &s
		}
	}

	traceID := entry.TraceID
	if traceID == "" {
		traceID = ""
	}

	_, err := l.db.Exec(
		`INSERT INTO app_logs (timestamp, level, module, message, data, trace_id) VALUES (?, ?, ?, ?, ?, ?)`,
		entry.Timestamp.UTC().Format(time.RFC3339),
		entry.Level,
		entry.Module,
		entry.Message,
		dataJSON,
		traceID,
	)
	if err != nil {
		log.Printf("[LOGGER] Failed to write log to DB: %v", err)
		return
	}

	// Auto-prune: keep only the 10,000 most recent rows
	l.db.Exec(`DELETE FROM app_logs WHERE id NOT IN (SELECT id FROM app_logs ORDER BY id DESC LIMIT 10000)`)
}

// writeOTel emits the log entry via OTel stdout exporter (through trace span on root tracer)
func (l *Logger) writeOTel(entry LogEntry) {
	ctx := context.Background()
	_, span := l.tracer.Start(ctx, "log/"+entry.Module,
		trace.WithAttributes(
			attribute.String("log.level", entry.Level),
			attribute.String("log.module", entry.Module),
			attribute.String("log.message", entry.Message),
		),
	)
	if entry.TraceID != "" {
		span.SetAttributes(attribute.String("log.trace_id", entry.TraceID))
	}
	if entry.Data != nil {
		if d, err := json.Marshal(entry.Data); err == nil {
			span.SetAttributes(attribute.String("log.data", string(d)))
		}
	}
	if entry.Level == LevelError {
		span.SetStatus(codes.Error, entry.Message)
	}
	span.End()
}

// Debugf logs at DEBUG level
func (l *Logger) Debugf(module, format string, args ...interface{}) {
	l.log(LevelDebug, module, fmt.Sprintf(format, args...), nil)
}

// DebugfWithData logs at DEBUG level with structured data
func (l *Logger) DebugfWithData(module, format string, data map[string]interface{}, args ...interface{}) {
	l.log(LevelDebug, module, fmt.Sprintf(format, args...), data)
}

// Infof logs at INFO level
func (l *Logger) Infof(module, format string, args ...interface{}) {
	l.log(LevelInfo, module, fmt.Sprintf(format, args...), nil)
}

// InfofWithData logs at INFO level with structured data
func (l *Logger) InfofWithData(module, format string, data map[string]interface{}, args ...interface{}) {
	l.log(LevelInfo, module, fmt.Sprintf(format, args...), data)
}

// Warnf logs at WARN level
func (l *Logger) Warnf(module, format string, args ...interface{}) {
	l.log(LevelWarn, module, fmt.Sprintf(format, args...), nil)
}

// WarnfWithData logs at WARN level with structured data
func (l *Logger) WarnfWithData(module, format string, data map[string]interface{}, args ...interface{}) {
	l.log(LevelWarn, module, fmt.Sprintf(format, args...), data)
}

// Errorf logs at ERROR level
func (l *Logger) Errorf(module, format string, args ...interface{}) {
	l.log(LevelError, module, fmt.Sprintf(format, args...), nil)
}

// ErrorfWithData logs at ERROR level with structured data
func (l *Logger) ErrorfWithData(module, format string, data map[string]interface{}, args ...interface{}) {
	l.log(LevelError, module, fmt.Sprintf(format, args...), data)
}

// Stop gracefully stops the logger
func (l *Logger) Stop() {
	l.once.Do(func() {
		close(l.logCh)
		<-l.done
	})
}
