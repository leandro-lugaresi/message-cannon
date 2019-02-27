package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/leandro-lugaresi/hub"
	"github.com/pkg/errors"
)

type httpRunner struct {
	client       *http.Client
	ignoreOutput bool
	hub          *hub.Hub
	url          string
	headers      map[string]string
	returnOn5xx  int
}

func (p *httpRunner) Process(ctx context.Context, msg Message) (int, error) {
	req, err := p.prepareRequest(msg)
	if err != nil {
		return ExitNACKRequeue, errors.Wrap(err, "request creation failed")
	}
	req = req.WithContext(ctx)
	resp, body, err := p.executeRequest(req)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return ExitTimeout, &Error{Err: netErr, StatusCode: -1}
		}
		return ExitNACKRequeue, errors.Wrap(err, "failed doing the request")
	}
	if resp.StatusCode >= 500 {
		return p.returnOn5xx, &Error{
			Err:        errors.New("receive an 5xx error from request"),
			StatusCode: resp.StatusCode,
			Output:     body,
		}
	}
	if resp.StatusCode >= 400 {
		return ExitNACKRequeue, &Error{
			Err:        errors.New("receive an 4xx error from request"),
			StatusCode: resp.StatusCode,
			Output:     body,
		}
	}
	if p.ignoreOutput {
		return ExitACK, nil
	}
	content := struct {
		ResponseCode int `json:"response-code"`
	}{}
	err = json.Unmarshal(body, &content)
	if err != nil && len(body) > 0 {
		return ExitNACKRequeue, &Error{
			Err:        err,
			StatusCode: resp.StatusCode,
			Output:     body,
		}
	}
	return content.ResponseCode, nil
}

func (p *httpRunner) prepareRequest(msg Message) (*http.Request, error) {
	contentReader := bytes.NewReader(msg.Body)
	req, err := http.NewRequest("POST", p.url, contentReader)
	if err != nil {
		return req, err
	}
	p.setHeaders(req, msg)
	return req, nil
}

func (p *httpRunner) setHeaders(req *http.Request, msg Message) {
	for k, v := range p.headers {
		req.Header.Set(k, v)
	}
	for k, v := range msg.Headers {
		switch vt := v.(type) {
		case int, int16, int32, int64, float32, float64:
			req.Header.Set(k, fmt.Sprint(vt))
		case string:
			req.Header.Set(k, vt)
		case []byte:
			req.Header.Set(k, string(vt))
		case time.Time:
			req.Header.Set(k, vt.Format(http.TimeFormat))
		case bool:
			req.Header.Set(k, strconv.FormatBool(vt))
		}
	}
}

func (p *httpRunner) executeRequest(req *http.Request) (*http.Response, []byte, error) {
	resp, err := p.client.Do(req)
	if err != nil {
		return resp, []byte{}, err
	}
	defer func() {
		deferErr := resp.Body.Close()
		if deferErr != nil {
			p.hub.Publish(hub.Message{
				Name:   "system.log.error",
				Body:   []byte("error closing the response body"),
				Fields: hub.Fields{"error": deferErr},
			})
		}
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		p.hub.Publish(hub.Message{
			Name:   "system.log.error",
			Body:   []byte("error reading the response body"),
			Fields: hub.Fields{"error": err},
		})
	}
	return resp, body, nil
}

func newHTTP(c Config, h *hub.Hub) *httpRunner {
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
	return &runner
}
