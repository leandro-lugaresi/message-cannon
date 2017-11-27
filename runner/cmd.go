package runner

import (
	"context"
	"os"
	"strings"
	"syscall"
	"time"

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
	Type         string        `mapstructure:"type"`
	IgnoreOutput bool          `mapstructure:"ignore-output"`
	Options      Options       `mapstructure:"options"`
	Timeout      time.Duration `mapstructure:"timeout"`
}

type command struct {
	cmd          string
	args         []string
	l            *zap.Logger
	ignoreOutput bool
}

func (c *command) Process(ctx context.Context, b []byte) int {
	cmd := exec.CommandContext(ctx, c.cmd, c.args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		c.l.Error("Receive an error creating the stdin pipe", zap.Error(err))
	}
	go func() {
		_, err := stdin.Write(b)
		if err != nil {
			c.l.Error("Failed writing to stdin", zap.Error(err))
		}
		err = stdin.Close()
		if err != nil {
			c.l.Error("Failed closing stdin", zap.Error(err))
		}
	}()
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.l.Error("Receive an error from command", zap.Error(err), zap.ByteString("output", output))
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}

		return ExitFailed
	}
	if !c.ignoreOutput && len(output) > 0 {
		c.l.Info("message processed with output", zap.ByteString("output", output))
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
		ignoreOutput: c.IgnoreOutput,
	}
	return &cmd, nil
}
