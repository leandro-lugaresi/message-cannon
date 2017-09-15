package runner

import (
	"bytes"
	"context"
	"syscall"

	"os/exec"

	"go.uber.org/zap"
)

type Runnable interface {
	Process(context.Context, []byte) error
}

type RunnerConfig struct {
	Type         string                 `mapstructure:"type"`
	IgnoreOutput bool                   `mapstructure:"ignore-output"`
	Options      map[string]interface{} `mapstructure:"options"`
}

type command struct {
	cmd          string
	args         []string
	stdErrLogger *logwriter
	stdOutLogger *logwriter
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
		c.stdErrLogger.log.Error("Receive an error from command", zap.Error(err))
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus()
			}
		}

		return 1
	}
}

func New(log *zap.Logger, c RunnerConfig) (Runnable, error) {

}
