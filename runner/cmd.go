package runner

import (
	"context"
	"os"
	"strings"
	"syscall"

	"os/exec"

	"github.com/leandro-lugaresi/message-cannon/event"
	"github.com/pkg/errors"
)

type command struct {
	cmd          string
	args         []string
	l            *event.Logger
	ignoreOutput bool
}

func (c *command) Process(ctx context.Context, b []byte) int {
	cmd := exec.CommandContext(ctx, c.cmd, c.args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		c.l.Error("Receive an error creating the stdin pipe", event.Field{"error", err})
	}
	go func() {
		_, pipeErr := stdin.Write(b)
		if pipeErr != nil {
			c.l.Error("Failed writing to stdin", event.Field{"error", pipeErr})
		}
		pipeErr = stdin.Close()
		if pipeErr != nil {
			c.l.Error("Failed closing stdin", event.Field{"error", pipeErr})
		}
	}()
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.l.Error("Receive an error from command", event.Field{"error", err}, event.Field{"output", output})
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}

		return ExitFailed
	}
	if !c.ignoreOutput && len(output) > 0 {
		c.l.Info("message processed with output", event.Field{"output", output})
	}
	return ExitACK
}

func newCommand(log *event.Logger, c Config) (*command, error) {
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
