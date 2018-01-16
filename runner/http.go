package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/leandro-lugaresi/message-cannon/event"
)

type httpRunner struct {
	client       *http.Client
	ignoreOutput bool
	log          *event.Logger
	url          string
	headers      map[string]string
	returnOn5xx  int
}

func (p *httpRunner) Process(ctx context.Context, b []byte) int {
	contentReader := bytes.NewReader(b)
	req, err := http.NewRequest("POST", p.url, contentReader)
	if err != nil {
		p.log.Error("error creating the request", event.Field{"error", err})
		return ExitRetry
	}
	for k, v := range p.headers {
		req.Header.Set(k, v)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Error("failed doing the request", event.Field{"error", err})
		return ExitRetry
	}
	defer func() {
		deferErr := resp.Body.Close()
		if deferErr != nil {
			p.log.Error("error closing the response body", event.Field{"error", deferErr})
		}
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		p.log.Error("error reading the response body", event.Field{"error", err})
	}
	if resp.StatusCode >= 500 {
		p.log.Error("receive an 5xx error from request",
			event.Field{"status-code", resp.StatusCode},
			event.Field{"output", body})
		return p.returnOn5xx
	}
	if resp.StatusCode >= 400 {
		p.log.Error("receive an 4xx error from request",
			event.Field{"status-code", resp.StatusCode},
			event.Field{"output", body})
		return ExitRetry
	}
	if p.ignoreOutput {
		return ExitACK
	}
	content := struct {
		ResponseCode int `json:"response-code"`
	}{}
	err = json.Unmarshal(body, &content)
	if err != nil && len(body) > 0 {
		p.log.Warn("failed to unmarshal the response",
			event.Field{"error", err},
			event.Field{"status-code", resp.StatusCode},
			event.Field{"output", body})
	}
	return content.ResponseCode
}

func newHTTP(log *event.Logger, c Config) (*httpRunner, error) {
	runner := httpRunner{
		log:          log,
		url:          c.Options.URL,
		ignoreOutput: c.IgnoreOutput,
		headers:      c.Options.Headers,
		returnOn5xx:  c.Options.ReturnOn5xx,
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
