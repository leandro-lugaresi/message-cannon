package rabbit

import (
	"testing"

	"github.com/streadway/amqp"
	"github.com/stretchr/testify/require"
)

func Test_assertRightArgsTypes(t *testing.T) {
	type args struct {
		args amqp.Table
	}
	tests := []struct {
		name string
		args args
		want amqp.Table
	}{
		{"int to int64", args{amqp.Table{"x-dead-letter-exchange": "", "x-message-ttl": int(3600000)}}, amqp.Table{"x-dead-letter-exchange": "", "x-message-ttl": int64(3600000)}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := assertRightTableTypes(tt.args.args)
			require.Exactly(t, tt.want, got)
		})
	}
}
