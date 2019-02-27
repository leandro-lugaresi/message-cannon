package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/creasty/defaults"
	"github.com/leandro-lugaresi/hub"
	"github.com/stretchr/testify/require"
)

type message struct {
	Sleep       time.Duration
	Code        int
	ContentType string
	Message     json.RawMessage
}

func Test_httpRunner_Process(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		msg := message{}
		if err := json.NewDecoder(req.Body).Decode(&msg); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
		if msg.Sleep > 0 {
			time.Sleep(msg.Sleep)
		}
		w.Header().Set("Content-Type", msg.ContentType)
		w.WriteHeader(msg.Code)
		_, err := w.Write([]byte(msg.Message))
		if err != nil {
			t.Fatal(err)
			return
		}
	})
	go func() {
		t.Error(http.ListenAndServe("localhost:8089", mux))
	}()
	time.Sleep(10 * time.Millisecond)

	type (
		args struct {
			b       []byte
			headers Headers
		}
		wants struct {
			exitCode int
			err      string
		}
	)
	tests := []struct {
		name         string
		args         args
		wants        wants
		ignoreOutput bool
	}{
		{
			"Success response without output",
			args{
				[]byte(`{"code":200, "contentType": "text/html", "message": ""}`),
				Headers{},
			},
			wants{
				ExitACK,
				"",
			},
			true,
		},
		{
			"Success response without output and ignoreOutput disabled",
			args{
				[]byte(`{"code":200, "contentType": "text/html", "message": ""}`),
				Headers{},
			},
			wants{
				ExitNACKRequeue,
				"json: cannot unmarshal string into Go value of type struct",
			},
			false,
		},
		{
			"Success response with output and ignoring the output",
			args{
				[]byte(`{"code":200, "contentType": "text/html", "message": "some random content here"}`),
				Headers{},
			},
			wants{
				ExitACK,
				"",
			},
			true,
		},
		{
			"200 with return-code",
			args{
				[]byte(`{"code":200, "contentType": "application/json", "message": {"response-code":0}}`),
				Headers{},
			},
			wants{
				ExitACK,
				"",
			},
			false,
		},
		{
			"200 with return-code",
			args{
				[]byte(`{"code":200, "contentType": "application/json", "message": {"response-code":1}}`),
				Headers{},
			},
			wants{
				ExitFailed,
				"",
			},
			false,
		},
		{
			"404 not found should NACK and requeue",
			args{
				[]byte(`{"code":404, "contentType": "text/html", "message": "some random content here"}`),
				Headers{},
			},
			wants{
				ExitNACKRequeue,
				"receive an 4xx error from request",
			},
			false,
		},
		{
			"request with error",
			args{
				[]byte(`{"code":500, "contentType": "text/html", "message": {"error": "PHP Exception :p"}}`),
				Headers{},
			},
			wants{
				ExitNACKRequeue,
				"receive an 5xx error from request",
			},
			false,
		},
		{
			"request with timeout",
			args{
				[]byte(`{"sleep": 4000000000, "code":500, "contentType": "text/html", "message": {"error": "Fooo"}}`),
				Headers{},
			},
			wants{
				ExitTimeout,
				"Client.Timeout exceeded while awaiting headers",
			},
			false,
		},
		{
			"request with headers",
			args{
				[]byte(`{"code":200, "contentType": "text/html", "message": {"response-code":0}, "returnHeaders": true}`),
				Headers{"Message-Id": 123456, "Content-Type": "Application/json"},
			},
			wants{
				ExitACK,
				"",
			},
			false,
		},
	}
	for _, tt := range tests {
		ctt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			config := Config{
				IgnoreOutput: ctt.ignoreOutput,
				Type:         "http",
				Timeout:      1 * time.Second,
				Options:      Options{URL: "http://localhost:8089"},
			}
			require.NoError(t, defaults.Set(&config))
			runner, err := New(config, hub.New())
			require.NoError(t, err)

			got, err := runner.Process(ctx, Message{Body: ctt.args.b, Headers: ctt.args.headers})
			if len(ctt.wants.err) > 0 {
				require.Contains(t, err.Error(), ctt.wants.err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, ctt.wants.exitCode, got, "result httpRunner.Process() differs")
		})
	}
}

func Test_httpRunner_setHeaders(t *testing.T) {
	tests := []struct {
		name    string
		msg     Message
		headers map[string]string
		want    http.Header
	}{
		{
			"headers from message should override headers from config",
			Message{Body: []byte(`{}`), Headers: Headers{"Authorization": "Basic from message"}},
			map[string]string{"Authorization": "Basic from config"},
			http.Header{
				"Authorization": []string{"Basic from message"},
			},
		},
		{
			"should convert integer types to string",
			Message{
				Body: []byte(`{}`),
				Headers: Headers{
					"test-int16": int16(1),
					"test-int32": int32(-111),
					"test-int64": int64(111),
					"test-int":   int(112),
				},
			},
			map[string]string{},
			http.Header{
				"Test-Int16": []string{"1"},
				"Test-Int32": []string{"-111"},
				"Test-Int64": []string{"111"},
				"Test-Int":   []string{"112"},
			},
		},
		{
			"should convert floats to string",
			Message{
				Body: []byte(`{}`),
				Headers: Headers{
					"test-float32": float32(-111.23),
					"test-float64": float64(111.23),
				},
			},
			map[string]string{},
			http.Header{
				"Test-Float32": []string{"-111.23"},
				"Test-Float64": []string{"111.23"},
			},
		},
		{
			"should convert bool to string",
			Message{
				Body: []byte(`{}`),
				Headers: Headers{
					"test-false": false,
					"test-true":  true,
				},
			},
			map[string]string{},
			http.Header{
				"Test-False": []string{"false"},
				"Test-True":  []string{"true"},
			},
		},
		{
			"should convert bytes to string",
			Message{
				Body:    []byte(`{}`),
				Headers: Headers{"test-bytes": []byte(`askjaskajsakjs`)},
			},
			map[string]string{},
			http.Header{"Test-Bytes": []string{"askjaskajsakjs"}},
		},
		{
			"should convert times to string",
			Message{
				Body:    []byte(`{}`),
				Headers: Headers{"test-date": time.Date(2019, time.March, 7, 10, 30, 0, 0, time.UTC)},
			},
			map[string]string{},
			http.Header{"Test-Date": []string{"Thu, 07 Mar 2019 10:30:00 GMT"}},
		},
	}
	for _, tt := range tests {
		ctt := tt
		t.Run(tt.name, func(t *testing.T) {
			p := newHTTP(Config{
				Options: Options{
					Headers: ctt.headers,
				},
			}, hub.New())
			req, err := http.NewRequest("POST", "http://localhost", bytes.NewReader(ctt.msg.Body))
			require.NoError(t, err)
			p.setHeaders(req, ctt.msg)
			require.Equal(t, ctt.want, req.Header)
		})
	}
}
