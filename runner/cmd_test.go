package runner

import (
	"context"
	"testing"
	"time"

	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/require"
)

func Test_command_Process(t *testing.T) {
	type (
		args struct {
			b       []byte
			timeout bool
		}
		wants struct {
			exitCode int
			err      string
		}
	)
	tests := []struct {
		name  string
		args  args
		wants wants
	}{
		{
			"Command with success",
			args{[]byte(`{"exitcode": 0, "delay": 100000, "info": "this is fine :)"}`), false},
			wants{
				ExitACK,
				"",
			},
		},
		{
			"Command with exit 1",
			args{[]byte(`{"exitcode": 1, "delay": 100000, "error": "Something is wrong :o"}`), false},
			wants{
				ExitFailed,
				"exit status 1",
			},
		},
		{
			"Command with php exception",
			args{[]byte(`{"delay": 2000000, "exception": "Something is wrong :o"}`), false},
			wants{
				255,
				"exit status 255",
			},
		},
		{
			"Command with timeout",
			args{[]byte(`{"exitcode": 0,"delay": 2000000}`), true},
			wants{
				-1,
				"signal: killed",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &command{
				cmd:  "testdata/receive.php",
				args: []string{},
				hub:  hub.New(),
			}
			ctx := context.Background()
			if tt.args.timeout {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
			}
			exitCode, err := c.Process(ctx, Message{Body: tt.args.b, Headers: map[string]string{}})
			if len(tt.wants.err) > 0 {
				require.Contains(t, err.Error(), tt.wants.err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wants.exitCode, exitCode, "command.Process wrong return value")
		})
	}
}
