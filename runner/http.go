package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	"github.com/leandro-lugaresi/hub"
)

type httpRunner struct {
	client       *http.Client
	ignoreOutput bool
	hub          *hub.Hub
	url          string
	headers      map[string]string
	returnOn5xx  int
}

func (p *httpRunner) Process(ctx context.Context, b []byte, headers map[string]string) int {
	contentReader := bytes.NewReader(b)
	req, err := http.NewRequest("POST", p.url, contentReader)
	if err != nil {
		p.hub.Publish(hub.Message{
			Name:   "runner.http.error",
			Body:   []byte("request creation failed"),
			Fields: hub.Fields{"error": err},
		})
		return ExitRetry
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for k, v := range p.headers {
		req.Header.Set(k, v)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		p.hub.Publish(hub.Message{
			Name:   "runner.http.error",
			Body:   []byte("failed doing the request"),
			Fields: hub.Fields{"error": err},
		})
		return ExitRetry
	}
	defer func() {
		deferErr := resp.Body.Close()
		if deferErr != nil {
			p.hub.Publish(hub.Message{
				Name:   "runner.http.error",
				Body:   []byte("error closing the response body"),
				Fields: hub.Fields{"error": deferErr},
			})
		}
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		p.hub.Publish(hub.Message{
			Name:   "runner.http.error",
			Body:   []byte("error reading the response body"),
			Fields: hub.Fields{"error": err},
		})
	}
	if resp.StatusCode >= 500 {
		p.hub.Publish(hub.Message{
			Name:   "runner.http.error",
			Body:   []byte("receive an 5xx error from request"),
			Fields: hub.Fields{"output": body, "status-code": resp.StatusCode},
		})
		return p.returnOn5xx
	}
	if resp.StatusCode >= 400 {
		p.hub.Publish(hub.Message{
			Name:   "runner.http.error",
			Body:   []byte("receive an 4xx error from request"),
			Fields: hub.Fields{"output": body, "status-code": resp.StatusCode},
		})
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
		p.hub.Publish(hub.Message{
			Name:   "runner.http.error",
			Body:   []byte("failed to unmarshal the response"),
			Fields: hub.Fields{"error": err, "output": body, "status-code": resp.StatusCode},
		})
		return ExitNACKRequeue
	}
	return content.ResponseCode
}

func newHTTP(c Config, h *hub.Hub) (*httpRunner, error) {
	runner := httpRunner{
		hub:          h,
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
