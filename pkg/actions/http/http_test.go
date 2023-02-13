package http

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/shiningrush/fastflow/pkg/entity/run"
	"github.com/stretchr/testify/mock"
)

func TestMain(m *testing.M) {
	go mockHTTPServer()
	time.Sleep(time.Second)
	os.Exit(m.Run())
}

const addr = "127.0.0.1:12345"

func TestHTTP_Run(t *testing.T) {
	ctx := &run.MockExecuteContext{}
	var (
		saveKey  string
		response *HTTPResponse
	)
	ctx.On("WithValue", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
		saveKey = args[0].(string)
		response = args[1].(*HTTPResponse)
	})

	ctx.On("Tracef", mock.Anything, mock.Anything).Return()
	ctx.On("Trace", mock.Anything, mock.Anything).Return()
	ctx.On("WithValue", mock.Anything, mock.Anything).Return()

	ctx.On("Context").Return(context.Background())

	type args struct {
		ctx    run.ExecuteContext
		params interface{}
	}
	url := "http://" + addr
	tests := []struct {
		name         string
		args         args
		wantSaveKey  string
		wantResponse *HTTPResponse
		wantErr      bool
	}{
		{
			name: "get",
			args: args{
				ctx: ctx,
				params: &HTTPParams{
					URL: url,
					Query: map[string]string{
						"q": "test",
					},
				},
			},
			wantSaveKey: "httpResponse",
			wantResponse: &HTTPResponse{Body: map[string]interface{}{
				"method": "GET",
				"query":  "q=test",
			}},
			wantErr: false,
		},
		{
			name: "post",
			args: args{
				ctx: ctx,
				params: &HTTPParams{
					Method: MethodPost,
					URL:    url,
					Body: map[string]interface{}{
						"a": 100,
					},
					RespSaveKey: "aa",
				},
			},
			wantSaveKey: "aa",
			wantResponse: &HTTPResponse{Body: map[string]interface{}{
				"method": "POST",
				"body":   `{"a":100}`,
			}},
			wantErr: false,
		},
		{
			name: "empty url",
			args: args{
				ctx: ctx,
				params: &HTTPParams{
					URL: "",
					Query: map[string]string{
						"q": "test",
					},
				},
			},
			wantSaveKey: "",
			wantErr:     true,
		},
		{
			name: "err url",
			args: args{
				ctx: ctx,
				params: &HTTPParams{
					Method: MethodGet,
					URL:    "err",
					Query: map[string]string{
						"q": "test",
					},
				},
			},
			wantSaveKey: "",
			wantErr:     true,
		},
		{
			name: "err status code",
			args: args{
				ctx: ctx,
				params: &HTTPParams{
					Method: MethodGet,
					URL:    url,
					Query: map[string]string{
						"q": "test",
					},
					RawBody: "error",
				},
			},
			wantSaveKey: "",
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &HTTP{}
			saveKey = ""
			response = nil
			err := h.Run(tt.args.ctx, tt.args.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				t.Logf("want err: %v", err)
			}
			if tt.wantSaveKey != "" {
				assert.Equal(t, tt.wantSaveKey, saveKey)
			}

			if tt.wantResponse != nil && response != nil {
				if response.Body != nil {
					assert.Equal(t, tt.wantResponse.Body, response.Body)
				} else {
					assert.Equal(t, tt.wantResponse.Raw, response.Raw)
				}
			} else {
				t.Log(response)
			}
		})
	}
}

func mockHTTPServer() {
	type response struct {
		Method string `json:"method,omitempty"`
		Query  string `json:"query,omitempty"`
		Body   string `json:"body,omitempty"`
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		bs, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Printf("Error reading body: %v", err)
		}
		resp := &response{
			Method: r.Method,
			Query:  r.URL.Query().Encode(),
			Body:   string(bs),
		}

		if string(bs) == "error" {
			w.WriteHeader(504)
		}

		bytes, _ := json.Marshal(resp)
		_, _ = w.Write(bytes)
	})

	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
