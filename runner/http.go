package runner

import (
	"bytes"
	"context"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type httpRunner struct {
	url          string
	headers      map[string]string
	l            *zap.Logger
	ignoreOutput bool
	client       *http.Client
}

func (p *httpRunner) Process(ctx context.Context, b []byte) int {
	contentReader := bytes.NewReader(b)
	req, err := http.NewRequest("POST", p.url, contentReader)
	if err != nil {
		p.l.Error("Error creating one request", zap.Error(err))
		return ExitRetry
	}
	for k, v := range p.headers {
		req.Header.Set(k, v)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		p.l.Error("Failed when on request", zap.Error(err))
		return ExitRetry
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			p.l.Error("Error closing the response body", zap.Error(err))
		}
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		p.l.Error("Error reading the request response body", zap.Error(err))
	}
	if resp.StatusCode >= 500 {
		p.l.Error("Receive an 5xx error from request",
			zap.Int("status-code", resp.StatusCode),
			zap.ByteString("output", body))
		return ExitNACKRequeue
	}
	if resp.StatusCode >= 400 {
		p.l.Error("Receive an 4xx error from request",
			zap.Int("status-code", resp.StatusCode),
			zap.ByteString("output", body))
		return ExitRetry
	}
	if !p.ignoreOutput && len(body) > 0 {
		p.l.Info("message processed with output",
			zap.Int("status-code", resp.StatusCode),
			zap.ByteString("output", body))
	}

	return ExitACK
}

func newHTTP(log *zap.Logger, c Config) (*httpRunner, error) {
	runner := httpRunner{
		l:            log,
		url:          c.Options.URL,
		ignoreOutput: c.IgnoreOutput,
		headers:      c.Options.Headers,
		client: &http.Client{
			Timeout: c.Timeout,
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout: 5 * time.Second,
				}).Dial,
				TLSHandshakeTimeout: 5 * time.Second,
			},
		},
	}
	return &runner, nil
}
