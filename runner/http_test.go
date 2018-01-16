package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/leandro-lugaresi/message-cannon/event"
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
	tests := []struct {
		name string
		want int
		msg  []byte
	}{
		{
			"Success response without output", ExitACK,
			[]byte(`{"code":200, "contentType": "text/html", "message": ""}`),
		},
		{
			"Success response with output", ExitACK,
			[]byte(`{"code":200, "contentType": "text/html", "message": "some random content here"}`),
		},
		{
			"200 with return-code", ExitNACK,
			[]byte(`{"code":200, "contentType": "application/json", "message": {"response-code":3}}`),
		},
		{
			"404 not found should retry", ExitRetry,
			[]byte(`{"code":404, "contentType": "text/html", "message": "some random content here"}`),
		},
		{
			"request with error", ExitNACKRequeue,
			[]byte(`{"code":500, "contentType": "text/html", "message": {"error": "PHP Exception :p"}}`),
		},
		{
			"request with timeout", ExitTimeout,
			[]byte(`{"sleep": 4000000000, "code":500, "contentType": "text/html", "message": {"error": "PHP Exception :p"}}`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := event.NewLogger(event.NewNoOpHandler(), 30)
			ctx := context.Background()
			runner, err := New(logger, Config{
				IgnoreOutput: false,
				Type:         "http",
				Timeout:      1 * time.Second,
				Options: Options{
					URL: "http://localhost:8089",
				},
			})
			require.NoError(t, err)

			got := runner.Process(ctx, tt.msg)

			logger.Close()
			require.Equal(t, tt.want, got, "result httpRunner.Process() differs")
		})
	}
}
