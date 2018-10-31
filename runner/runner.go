package runner

import (
	"context"
	"strings"
	"time"

	"github.com/leandro-lugaresi/hub"
	"github.com/pkg/errors"
)

// Exit constants used to know how handle the message.
// The consumer runnig is the responsible to understand this status and handle them properly.
const (
	ExitTimeout     = -1
	ExitACK         = 0
	ExitFailed      = 1
	ExitNACK        = 3
	ExitNACKRequeue = 4
	ExitRetry       = 5
)

type (
	// Runnable represent an runnable used by consumers to handle messages.
	Runnable interface {
		Process(context.Context, Message) (int, error)
	}

	// Message is an container struct with general fields used by messages.
	Message struct {
		Body    []byte
		Headers map[string]string
	}

	// Options is a composition os all options used internally by runners.
	// options not needed by one runner will be ignored.
	Options struct {
		// Command options
		Path string   `mapstructure:"path"`
		Args []string `mapstructure:"args"`
		// HTTP options
		URL         string            `mapstructure:"url"`
		ReturnOn5xx int               `mapstructure:"return-on-5xx" default:"4"`
		Headers     map[string]string `mapstructure:"headers" default:"{}"`
	}

	// Config is an composition of options and configurations used by this runnables.
	Config struct {
		Type         string        `mapstructure:"type"`
		IgnoreOutput bool          `mapstructure:"ignore-output"`
		Options      Options       `mapstructure:"options"`
		Timeout      time.Duration `mapstructure:"timeout"`
	}

	// Error describes an error during the Process phase.
	Error struct {
		Err        error
		StatusCode int
		Output     []byte
	}
)

// New create and return a Runnable based on the config type. if the type didn't exist an error is returned.
func New(c Config, h *hub.Hub) (Runnable, error) {
	switch c.Type {
	case "command":
		return newCommand(c, h)
	case "http":
		return newHTTP(c, h)
	}
	return nil, errors.Errorf(
		"Invalid Runner type (\"%s\") expecting one of (%s)",
		c.Type,
		strings.Join([]string{"command", "http"}, ", "))
}

func (e *Error) Error() string {
	return e.Err.Error()
}
