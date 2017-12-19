package events

import (
	"context"
	"log"

	gendiodes "code.cloudfoundry.org/go-diodes"
)

// Level defines log levels used internally.
type Level uint8

// Field represent one key-value field passed with the log message.
type Field struct {
	key   string
	value interface{}
}

// Message represet one log message.
type Message struct {
	Level  Level
	Msg    string
	Fields []Field
}

// Handler is the function used to process every message.
type Handler func(msg Message)

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
		c.handler(msg)
	}
	c.done <- struct{}{}
}

// NewLogger returns a new Logger to be used to log messages.
func NewLogger(handler Handler, cap int) *Logger {
	ctx, cancelFunc := context.WithCancel(context.Background())
	l := &Logger{
		core: &core{
			handler: handler,
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
