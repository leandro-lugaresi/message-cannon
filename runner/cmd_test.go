package runner

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
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
			[]string{`"level":"error","msg":"Receive an error from command","error":"exit status 1","output":"Something is wrong :o"`},
		},
		{
			"Command with php exception",
			args{[]byte(`{"delay": 2000000, "exception": "Something is wrong :o"}`), false},
			255,
			[]string{
				`"level":"error","msg":"Receive an error from command","error":"exit status 255","output":"PHP Fatal error:`,
			},
		},
		{
			"Command with timeout",
			args{[]byte(`{"exitcode": 0,"delay": 2000000}`), true},
			-1,
			[]string{
				`"level":"error","msg":"Receive an error from command","error":"signal: killed"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			logger := zap.NewExample()
			c := &command{
				cmd:  "testdata/receive.php",
				args: []string{},
				l:    logger,
			}
			ctx := context.Background()
			if tt.args.timeout {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 100*time.Millisecond)
				defer cancel()
			}
			if got := c.Process(ctx, tt.args.b); got != tt.want {
				t.Errorf("command.Process() = %v, want %v", got, tt.want)
			}
			err := w.Close()
			if err != nil {
				t.Fatal(err, "failed to close the pipe writer")
			}
			out, _ := ioutil.ReadAll(r)
			for _, entry := range tt.logEntries {
				assert.Contains(t, string(out), entry, "")
			}
			os.Stdout = originalStdout
		})
	}
}

func TestNew(t *testing.T) {
	logger := zap.NewNop()
	tests := []struct {
		name       string
		c          Config
		want       Runnable
		wantErr    bool
		errMessage string
	}{
		{"With undefined type", Config{Type: "invalid-c3"}, nil, true, "Invalid Runner type (\"invalid-c3\") expecting (command)"},
		{
			"With command type but with executable not found",
			Config{
				Type:    "command",
				Options: Options{Path: "/bin/fooo"},
			},
			(Runnable)(nil), true, "The command /bin/fooo didn't exist",
		},
		{
			"With an valid command",
			Config{
				Type:         "command",
				Options:      Options{Path: "/usr/bin/tail -f"},
				IgnoreOutput: true,
			},
			&command{
				cmd:          "/usr/bin/tail",
				args:         []string{"-f"},
				l:            logger,
				ignoreOutput: true,
			}, false, "",
		},
		{
			"With an valid command and did not ignore output",
			Config{
				Type:         "command",
				Options:      Options{Path: "testdata/receive.php"},
				IgnoreOutput: false,
			},
			&command{
				cmd: "testdata/receive.php",
				l:   logger,
			}, false, "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(zap.NewNop(), tt.c)
			assert.Equal(t, tt.wantErr, (err != nil), "New() error = %v, wantErr %v", err, tt.wantErr)
			if err != nil && tt.wantErr {
				assert.EqualValues(t, tt.errMessage, err.Error(), "Error message is different than expected")
			}
			if tt.want == nil {
				assert.Nil(t, got, "Runnable returned must be nil")
			} else {
				assert.Equal(t, tt.want, got, "")
			}
		})
	}
}
