package runner

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type logwriter struct {
	log   *zap.Logger
	level zapcore.Level
}

func (lwr *logwriter) Write(p []byte) (n int, err error) {
	if ce := lwr.log.Check(lwr.level, ""); ce != nil {
		ce.Write(zap.ByteString("output", p))
	}
	return len(p), nil
}
