package cmd

import (
	"context"
	"io"
)

// Runner encapsulates what is done for every message received.
type Runner func(context.Context, io.Writer, []byte) error

// Consumer consume messages and pass to .
type Consumer interface {
	// TODO: Create the state, we will add some metrics here
	// State returns a copy of the executor's current operation state.
	// State() State

	// Run will get the messages and pass to the runner.
	Run(context.Context, Runner) error
}

type manager struct {
	consumers []Consumer
}
