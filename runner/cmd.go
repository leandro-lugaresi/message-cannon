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
	cmd  string
	args []string
	hub  *hub.Hub
}

func (c *command) Process(ctx context.Context, msg Message) (int, error) {
	cmd := exec.CommandContext(ctx, c.cmd, c.args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return ExitNACKRequeue, errors.Wrap(err, "open pipe to stdin failed")
	}
	go func() {
		_, pipeErr := stdin.Write(msg.Body)
		if pipeErr != nil {
			c.hub.Publish(hub.Message{
				Name:   "system.log.error",
				Body:   []byte("failed writing to stdin"),
				Fields: hub.Fields{"error": pipeErr},
			})
		}
		pipeErr = stdin.Close()
		if pipeErr != nil {
			c.hub.Publish(hub.Message{
				Name:   "system.log.error",
				Body:   []byte("close stdin failed"),
				Fields: hub.Fields{"error": pipeErr},
			})
		}
	}()
	output, err := cmd.CombinedOutput()
	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus(), &Error{
					Err:        exiterr,
					Output:     output,
					StatusCode: status.ExitStatus(),
				}
			}
		}
		return ExitNACKRequeue, err
	}
	return ExitACK, nil
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
		cmd:  c.Options.Path,
		args: c.Options.Args,
		hub:  h,
	}
	return &cmd, nil
}
