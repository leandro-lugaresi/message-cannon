package events

import (
	"fmt"
	"sync"
)

// Level defines log levels.
type Level uint8

type Handler func(msg Message)

type Field struct {
	key   string
	value interface{}
}

// Logger defines the logger
type Logger struct {
	handler Handler
	fields  []Field
	events  chan Message
	wg      sync.WaitGroup
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

func NewLogger(handler Handler, cap int) *Logger {
	l := &Logger{
		events:  make(chan Message, cap),
		handler: handler,
	}
	l.wg.Add(1)
	go func(log *Logger) {
		for msg := range log.events {
			log.handler(msg)
		}
		log.wg.Done()
	}(l)
	return l
}

type Message struct {
	Level  Level
	Msg    string
	Fields []Field
}

func (l *Logger) Log(level Level, msg string, fields ...Field) {
	fields = append(fields, l.fields...)
	m := Message{
		Level:  level,
		Msg:    msg,
		Fields: fields,
	}
	select {
	case l.events <- m:
	default:
		fmt.Println("Message dropped")
	}
}

func (l *Logger) With(fields ...Field) *Logger {
	log := *l
	log.fields = append(log.fields, fields...)
	return &log
}

func (l *Logger) Close() {
	close(l.events)
	l.wg.Wait()
}
