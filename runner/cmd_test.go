package runner

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/leandro-lugaresi/message-cannon/event"
	"github.com/stretchr/testify/assert"
)

func Test_command_Process(t *testing.T) {
	type args struct {
		b       []byte
		timeout bool
	}
	tests := []struct {
		name       string
		args       args
		want       int
		logEntries []string
	}{
		{
			"Command with success",
			args{[]byte(`{"exitcode": 0, "delay": 100000, "info": "this is fine :)"}`), false},
			0,
			[]string{`"output":"this is fine :)"`},
		},
		{
			"Command with exit 1",
			args{[]byte(`{"exitcode": 1, "delay": 100000, "error": "Something is wrong :o"}`), false},
			1,
			[]string{`"level":"error","error":"exit status 1","output":"Something is wrong :o","message":"receive an error from command"}`},
		},
		{
			"Command with php exception",
			args{[]byte(`{"delay": 2000000, "exception": "Something is wrong :o"}`), false},
			255,
			[]string{
				`"level":"error","error":"exit status 255","output":"PHP Fatal error:`,
				`"message":"receive an error from command"`,
			},
		},
		{
			"Command with timeout",
			args{[]byte(`{"exitcode": 0,"delay": 2000000}`), true},
			-1,
			[]string{
				`"level":"error","error":"signal: killed","output":"","message":"receive an error from command"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, err := os.Pipe()
			require.NoError(t, err, "Fail creating the pipe file")
			logger := event.NewLogger(event.NewZeroLogHandler(w, false), 30)
			c := &command{
				cmd:  "testdata/receive.php",
				args: []string{},
				log:  logger,
			}
			ctx := context.Background()
			if tt.args.timeout {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
			}
			if got := c.Process(ctx, tt.args.b, map[string]string{}); got != tt.want {
				t.Errorf("command.Process() = %v, want %v", got, tt.want)
			}
			logger.Close()

			err = w.Close()
			if err != nil {
				t.Fatal(err, "failed to close the pipe writer")
			}
			out, _ := ioutil.ReadAll(r)
			for _, entry := range tt.logEntries {
				assert.Contains(t, string(out), entry, "")
			}
		})
	}
}
