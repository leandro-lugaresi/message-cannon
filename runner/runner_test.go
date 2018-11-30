package runner

import (
	"testing"

	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name       string
		c          Config
		want       Runnable
		wantErr    bool
		errMessage string
	}{
		{
			"With undefined type",
			Config{Type: "invalid-c3"},
			nil,
			true,
			"Invalid Runner type (\"invalid-c3\") expecting one of (command, http)",
		},
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
				cmd:  "/usr/bin/tail",
				args: []string{"-f"},
				hub:  hub.New(),
			}, false, "",
		},
		{
			"With an valid command and did not ignore output",
			Config{
				Type:    "command",
				Options: Options{Path: "testdata/receive.php"},
			},
			&command{
				cmd: "testdata/receive.php",
				hub: hub.New(),
			}, false, "",
		},
	}
	for _, tt := range tests {
		ctt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := New(ctt.c, hub.New())
			assert.Equal(t, ctt.wantErr, (err != nil), "New() error = %v, wantErr %v", err, ctt.wantErr)
			if err != nil && ctt.wantErr {
				assert.EqualValues(t, ctt.errMessage, err.Error(), "Error message is different than expected")
			}
			if ctt.want == nil {
				assert.Nil(t, got, "Runnable returned must be nil")
			} else {
				assert.Equal(t, ctt.want, got, "")
			}
		})
	}
}
