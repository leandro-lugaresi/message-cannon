package runner

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
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
		t.Error(http.ListenAndServe("localhost:8080", mux))
	}()

	tests := []struct {
		name       string
		want       int
		msg        []byte
		logEntries []string
	}{
		{
			"Success response without output", ExitACK,
			[]byte(`{"code":200, "contentType": "text/html", "message": ""}`),
			[]string{},
		},
		{
			"Success response with output", ExitACK,
			[]byte(`{"code":200, "contentType": "text/html", "message": "some random content here"}`),
			[]string{`{"level":"info","msg":"message processed with output","status-code":200,"output":"\"some random content here\""}`},
		},
		{
			"404 not found should retry", ExitRetry,
			[]byte(`{"code":404, "contentType": "text/html", "message": "some random content here"}`),
			[]string{`{"level":"error","msg":"Receive an 4xx error from request","status-code":404,"output":"\"some random content here\""}`},
		},
		{
			"request with error", ExitNACKRequeue,
			[]byte(`{"code":500, "contentType": "text/html", "message": {"error": "PHP Exception :p"}}`),
			[]string{`{"level":"error","msg":"Receive an 5xx error from request","status-code":500,"output":"{\"error\": \"PHP Exception :p\"}"}`},
		},
		{
			"request with timeout", ExitRetry,
			[]byte(`{"sleep": 4000000000, "code":500, "contentType": "text/html", "message": {"error": "PHP Exception :p"}}`),
			[]string{`{"level":"error","msg":"Failed when on request","error":"Post http://localhost:8080: net/http: request canceled (Client.Timeout exceeded while awaiting headers)"}`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			logger := zap.NewExample()
			ctx := context.Background()
			runner, err := New(logger, Config{
				IgnoreOutput: false,
				Type:         "http",
				Timeout:      1 * time.Second,
				Options: Options{
					URL: "http://localhost:8080",
				},
			})
			require.NoError(t, err)

			got := runner.Process(ctx, tt.msg)

			err = w.Close()
			if err != nil {
				t.Fatal(err, "failed to close the pipe writer")
			}
			out, _ := ioutil.ReadAll(r)
			for _, entry := range tt.logEntries {
				require.Contains(t, string(out), entry, "")
			}
			os.Stdout = originalStdout
			require.Equal(t, tt.want, got, "result httpRunner.Process() differs")
		})
	}
}
