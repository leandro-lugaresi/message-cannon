package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

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
			http.Error(w, http.StatusText(405), 405)
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
			headers map[string]string
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
				map[string]string{},
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
				map[string]string{},
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
				map[string]string{},
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
				map[string]string{},
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
				map[string]string{},
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
				map[string]string{},
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
				map[string]string{},
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
				[]byte(`{"sleep": 4000000000, "code":500, "contentType": "text/html", "message": {"error": "PHP Exception :p"}}`),
				map[string]string{},
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
				map[string]string{"Message-Id": "123456", "Content-Type": "Application/json"},
			},
			wants{
				ExitACK,
				"",
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			runner, err := New(Config{
				IgnoreOutput: tt.ignoreOutput,
				Type:         "http",
				Timeout:      1 * time.Second,
				Options:      Options{URL: "http://localhost:8089"},
			}, hub.New())
			require.NoError(t, err)

			got, err := runner.Process(ctx, Message{Body: tt.args.b, Headers: tt.args.headers})
			if len(tt.wants.err) > 0 {
				require.Contains(t, err.Error(), tt.wants.err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tt.wants.exitCode, got, "result httpRunner.Process() differs")
		})
	}
}
