package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

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

func TestRunShoudSetDefaults(t *testing.T) {
	t.Run("with empty values", func(t *testing.T) {
		got, err := New(zap.NewNop(), Config{
			Type: "http",
		})
		assert.NoError(t, err)
		httpRunner := got.(*httpRunner)
		assert.Equal(t, false, httpRunner.ignoreOutput)
		assert.Equal(t, 4, httpRunner.returnOn5xx)
	})
	t.Run("using ack integer should overide to default", func(t *testing.T) {
		got, err := New(zap.NewNop(), Config{
			Type: "http",
			Options: Options{
				ReturnOn5xx: ExitACK,
			},
		})
		assert.NoError(t, err)
		httpRunner := got.(*httpRunner)
		assert.Equal(t, 4, httpRunner.returnOn5xx)
	})
}
