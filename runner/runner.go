package runner

import (
	"context"
	"strings"
	"time"

	"github.com/leandro-lugaresi/message-cannon/event"
	"github.com/pkg/errors"
	"gopkg.in/mcuadros/go-defaults.v1"
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

// Runnable represent an runnable used by consumers to handle messages.
type Runnable interface {
	Process(context.Context, []byte) int
}

// Options is a composition os all options used internally by runners.
// options not needed by one runner will be ignored.
type Options struct {
	// Command options
	Path string   `mapstructure:"path"`
	Args []string `mapstructure:"args"`
	// HTTP options
	URL         string `mapstructure:"url"`
	ReturnOn5xx int    `mapstructure:"return-on-5xx" default:"4"`
	Headers     map[string]string
}

// Config is an composition of options and configurations used by this runnables.
type Config struct {
	Type         string        `mapstructure:"type"`
	IgnoreOutput bool          `mapstructure:"ignore-output"`
	Options      Options       `mapstructure:"options"`
	Timeout      time.Duration `mapstructure:"timeout"`
}

// New create and return a Runnable based on the config type. if the type didn't exist an error is returned.
func New(log *event.Logger, c Config) (Runnable, error) {
	defaults.SetDefaults(&c)
	defaults.SetDefaults(&c.Options)
	switch c.Type {
	case "command":
		return newCommand(log, c)
	case "http":
		return newHTTP(log, c)
	}
	return nil, errors.Errorf(
		"Invalid Runner type (\"%s\") expecting (%s)",
		c.Type,
		strings.Join([]string{"command"}, ", "))
}
