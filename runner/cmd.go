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
	log          *event.Logger
	ignoreOutput bool
}

func (c *command) Process(ctx context.Context, b []byte, headers map[string]string) int {
	cmd := exec.CommandContext(ctx, c.cmd, c.args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		c.log.Error("receive an error creating the stdin pipe", event.KV("error", err))
	}
	go func() {
		_, pipeErr := stdin.Write(b)
		if pipeErr != nil {
			c.log.Error("failed writing to stdin", event.KV("error", pipeErr))
		}
		pipeErr = stdin.Close()
		if pipeErr != nil {
			c.log.Error("failed closing stdin", event.KV("error", pipeErr))
		}
	}()
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.log.Error("receive an error from command", event.KV("error", err), event.KV("output", output))
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}

		return ExitFailed
	}
	if !c.ignoreOutput && len(output) > 0 {
		c.log.Info("message processed with output", event.KV("output", output))
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
		log:          log,
		ignoreOutput: c.IgnoreOutput,
	}
	return &cmd, nil
}
