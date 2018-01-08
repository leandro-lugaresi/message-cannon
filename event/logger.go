package event

import (
	"context"
	"log"

	gendiodes "code.cloudfoundry.org/go-diodes"
)

// Level defines log levels used internally.
type Level uint8

// Field represent one key-value field passed with the log message.
type Field struct {
	Key   string
	Value interface{}
}

// Message represet one log message.
type Message struct {
	Level  Level
	Msg    string
	Fields []Field
}

// Handler represent and Logger handler.
// This handler is responsible to send the every log registry to somewhere.
type Handler interface {
	// Handle will send the message to somewhere, this function didn't expect errors.
	Handle(msg Message)
	// Sync will flush the buffers, if any.
	Sync()
}

// HandlerFunc is the function used to process every message.
type HandlerFunc func(msg Message)

func (h HandlerFunc) Handle(msg Message) {
	h(msg)
}
func (h HandlerFunc) Sync() {}

type core struct {
	handler Handler
	diode   *manyToOneMessage
	cancel  context.CancelFunc
	done    chan struct{}
}

// Logger defines the logger struct used to send logs to an Handler.
type Logger struct {
	fields []Field
	core   *core
}

const (
	// DebugLevel defines debug log level.
	DebugLevel Level = iota
	// InfoLevel defines info log level.
	InfoLevel
	// WarnLevel defines warn log level.
	WarnLevel
	// ErrorLevel defines error log level.
	ErrorLevel
	// FatalLevel defines fatal log level.
	FatalLevel
	// PanicLevel defines panic log level.
	PanicLevel
)

func (c *core) handle() {
	for {
		msg, closed := c.diode.Next()
		if closed {
			break
		}
		c.handler.Handle(msg)
	}
	c.handler.Sync()
	c.done <- struct{}{}
}

// NewLogger returns a new Logger to be used to log messages.
func NewLogger(cap int, options ...func(*Logger)) *Logger {
	ctx, cancelFunc := context.WithCancel(context.Background())
	l := &Logger{
		core: &core{
			handler: NewNoOpHandler(),
			diode: newManyToOneMessage(
				ctx,
				cap,
				gendiodes.AlertFunc(func(missed int) {
					log.Printf("Dropped %d messages from log", missed)
				})),
			cancel: cancelFunc,
			done:   make(chan struct{}, 1),
		},
	}
	for _, option := range options {
		option(l)
	}
	go l.core.handle()
	return l
}

// Log will send the message in a non blocking way.
func (l *Logger) Log(level Level, msg string, fields ...Field) {
	fields = append(fields, l.fields...)
	m := Message{
		Level:  level,
		Msg:    msg,
		Fields: fields,
	}
	l.core.diode.Set(m)
}

// With return a new sub Logger with fields attached.
func (l *Logger) With(fields ...Field) *Logger {
	log := *l
	log.fields = append(log.fields, fields...)
	return &log
}

// Close will close the logger.
func (l *Logger) Close() {
	l.core.cancel()
	<-l.core.done
}

// Debug will send an Debug event.
func (l *Logger) Debug(msg string, fields ...Field) {
	l.Log(DebugLevel, msg, fields...)
}

// Info will send an Info event.
func (l *Logger) Info(msg string, fields ...Field) {
	l.Log(InfoLevel, msg, fields...)
}

// Warn will send an Warning event.
func (l *Logger) Warn(msg string, fields ...Field) {
	l.Log(WarnLevel, msg, fields...)
}

// Error will send an Error event.
func (l *Logger) Error(msg string, fields ...Field) {
	l.Log(ErrorLevel, msg, fields...)
}

// NewNoOpHandler return an handler with always succeed without doing anything.
func NewNoOpHandler() Handler {
	return HandlerFunc(func(msg Message) {})
}
