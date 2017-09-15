package runner

import (
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Test_logwriter_Write(t *testing.T) {
	type fields struct {
		log   *zap.Logger
		level zapcore.Level
	}
	type args struct {
		p []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantN   int
		wantErr bool
	}{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lwr := &logwriter{
				log:   tt.fields.log,
				level: tt.fields.level,
			}
			gotN, err := lwr.Write(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("logwriter.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotN != tt.wantN {
				t.Errorf("logwriter.Write() = %v, want %v", gotN, tt.wantN)
			}
		})
	}
}
