package runner

import (
	"context"
	"os"
	"strings"
	"syscall"

	"os/exec"

	"github.com/leandro-lugaresi/hub"
	"github.com/pkg/errors"
)

type command struct {
	cmd          string
	args         []string
	hub          *hub.Hub
	ignoreOutput bool
}

func (c *command) Process(ctx context.Context, b []byte, headers map[string]string) int {
	cmd := exec.CommandContext(ctx, c.cmd, c.args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		c.hub.Publish(hub.Message{
			Name:   "command.process.error",
			Body:   []byte("open pipe to stdin failed"),
			Fields: hub.Fields{"error": err},
		})
	}
	go func() {
		_, pipeErr := stdin.Write(b)
		if pipeErr != nil {
			c.hub.Publish(hub.Message{
				Name:   "command.process.error",
				Body:   []byte("failed writing to stdin"),
				Fields: hub.Fields{"error": pipeErr},
			})
		}
		pipeErr = stdin.Close()
		if pipeErr != nil {
			c.hub.Publish(hub.Message{
				Name:   "command.process.error",
				Body:   []byte("close stdin failed"),
				Fields: hub.Fields{"error": pipeErr},
			})
		}
	}()
	output, err := cmd.CombinedOutput()
	if err != nil {
		c.hub.Publish(hub.Message{
			Name:   "runner.command.error",
			Body:   []byte("command exec failed"),
			Fields: hub.Fields{"error": err, "output": output},
		})
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}

		return ExitFailed
	}
	if !c.ignoreOutput && len(output) > 0 {
		c.hub.Publish(hub.Message{
			Name:   "runner.command.info",
			Body:   []byte("message processed with output"),
			Fields: hub.Fields{"output": output},
		})
	}
	return ExitACK
}

func newCommand(c Config, h *hub.Hub) (*command, error) {
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
		hub:          h,
		ignoreOutput: c.IgnoreOutput,
	}
	return &cmd, nil
}
