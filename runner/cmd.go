package runner

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"syscall"

	"os/exec"

	"github.com/pkg/errors"
	"go.uber.org/zap"
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

type Options struct {
	Path string   `mapstructure:"path"`
	Args []string `mapstructure:"args"`
}

// Config is an composition of all options and configurations used by this runnables.
type Config struct {
	Type         string  `mapstructure:"type"`
	IgnoreOutput bool    `mapstructure:"ignore-output"`
	Options      Options `mapstructure:"options"`
}

type command struct {
	cmd          string
	args         []string
	stdErrLogger io.Writer
	stdOutLogger io.Writer
	l            *zap.Logger
}

func (c *command) Process(ctx context.Context, b []byte) int {
	cmd := exec.CommandContext(ctx, c.cmd, c.args...)
	var bf bytes.Buffer
	bf.Write(b)
	cmd.Stdin = &bf
	cmd.Stderr = c.stdErrLogger
	cmd.Stdout = c.stdOutLogger
	err := cmd.Run()

	if err != nil {
		c.l.Error("Receive an error from command", zap.Error(err))
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}

		return ExitFailed
	}
	return ExitACK
}

// New create and return a Runnable based on the config type. if the type didn't exist an error is returned.
func New(log *zap.Logger, c Config) (Runnable, error) {
	switch c.Type {
	case "command":
		return newCommand(log, c)
	}
	return nil, errors.Errorf(
		"Invalid Runner type (\"%s\") expecting (%s)",
		c.Type,
		strings.Join([]string{"command"}, ", "))
}

func newCommand(log *zap.Logger, c Config) (*command, error) {
	if split := strings.Split(c.Options.Path, " "); len(split) > 1 {
		c.Options.Path = split[0]
		c.Options.Args = append(split[1:], c.Options.Args...)
	}
	if _, err := os.Stat(c.Options.Path); os.IsNotExist(err) {
		return nil, errors.Errorf("The command %s didn't exist", c.Options.Path)
	}
	cmd := command{
		cmd:          c.Options.Path,
		args:         c.Options.Args,
		l:            log,
		stdErrLogger: ioutil.Discard,
		stdOutLogger: ioutil.Discard,
	}
	if !c.IgnoreOutput {
		cmd.stdErrLogger = &logwriter{
			level: zap.ErrorLevel,
			log:   log,
		}
		cmd.stdOutLogger = &logwriter{
			level: zap.InfoLevel,
			log:   log,
		}
	}
	return &cmd, nil
}
