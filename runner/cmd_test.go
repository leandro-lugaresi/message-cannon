package runner

import (
	"context"
	"testing"
	"time"

	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/assert"
)

func Test_command_Process(t *testing.T) {
	type args struct {
		b       []byte
		timeout bool
	}
	tests := []struct {
		name string
		args args
		want int
		msgs []hub.Message
	}{
		{
			"Command with success",
			args{[]byte(`{"exitcode": 0, "delay": 100000, "info": "this is fine :)"}`), false},
			0,
			[]hub.Message{{
				Name:   "runner.command.info",
				Body:   []byte("message processed with output"),
				Fields: hub.Fields{"output": []byte("this is fine :)")},
			}},
		},
		{
			"Command with exit 1",
			args{[]byte(`{"exitcode": 1, "delay": 100000, "error": "Something is wrong :o"}`), false},
			1,
			[]hub.Message{{}},
		},
		{
			"Command with php exception",
			args{[]byte(`{"delay": 2000000, "exception": "Something is wrong :o"}`), false},
			255,
			[]hub.Message{{}},
		},
		{
			"Command with timeout",
			args{[]byte(`{"exitcode": 0,"delay": 2000000}`), true},
			-1,
			[]hub.Message{{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := hub.New()
			subs := h.Subscribe(10, "*.*.*")
			c := &command{
				cmd:  "testdata/receive.php",
				args: []string{},
				hub:  h,
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
			h.Close()

			for _, msg := range tt.msgs {
				assert.Equal(t, msg, <-subs.Receiver)
			}
		})
	}
}
