package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ActionHTTP = "http"

	DefaultTimeout             float64 = 10 * 60 // 10 minutes
	DefaultResponseContentType         = "application/json"
	DefaultResponseSaveKey             = "httpResponse"
	DefaultResponseHandler             = ResponseHandlerJSON
)

type Method string

const (
	MethodGet     Method = "GET"
	MethodHead    Method = "HEAD"
	MethodPost    Method = "POST"
	MethodPut     Method = "PUT"
	MethodPatch   Method = "PATCH" // RFC 5789
	MethodDelete  Method = "DELETE"
	MethodConnect Method = "CONNECT"
	MethodOptions Method = "OPTIONS"
	MethodTrace   Method = "TRACE"
)

const (
	HeaderContentTypeKey = "content-type"

	ContentTypeJSON = "application/json"
)

type ResponseHandler string

const (
	ResponseHandlerJSON = "json"
	ResponseHandlerNone = "none"
	ResponseHandlerXML  = "xml"
)

type HTTPParams struct {
	Method Method            `yaml:"method" json:"method"`
	URL    string            `yaml:"url" json:"url"`
	Path   string            `yaml:"path" json:"path"`
	Query  map[string]string `yaml:"query" json:"query"`
	// Body  结构化的 body, 发送请求会用 json 序列化
	Body    interface{} `yaml:"body" json:"body"`
	RawBody string      `yaml:"rawBody" json:"rawBody"`
	Header  http.Header `yaml:"header" json:"header"`

	// Client Options
	TimeoutSec float64 `yaml:"timeoutSec" json:"timeoutSec"`
	// Resp Options
	UseNumber   bool            `yaml:"useNumber" json:"useNumber"`
	RespHandler ResponseHandler `yaml:"responseHandler" json:"responseHandler"`
	RespSaveKey string          `yaml:"respKey" json:"respKey"`
}

func (p *HTTPParams) validate() error {
	if p.URL == "" {
		return fmt.Errorf("url cannot be empty")
	}

	if p.TimeoutSec == 0 {
		p.TimeoutSec = DefaultTimeout
	}

	if p.Method == "" {
		p.Method = MethodGet
	}
	if p.Header == nil {
		p.Header = http.Header{}
	}
	return nil
}

func (p *HTTPParams) getURL() string {
	urlBuilder := strings.Builder{}
	urlBuilder.WriteString(p.URL)
	if len(p.Path) > 0 {
		if (!strings.HasSuffix(p.URL, "/")) && (!strings.HasPrefix(p.Path, "/")) {
			urlBuilder.WriteString("/")
		}
		urlBuilder.WriteString(p.Path)
	}
	if len(p.Query) != 0 {
		query := url.Values{}
		for k, v := range p.Query {
			query.Add(k, v)
		}
		urlBuilder.WriteString("?")
		urlBuilder.WriteString(query.Encode())
	}
	return urlBuilder.String()
}

func (p *HTTPParams) buildRequest(ctx context.Context) (*http.Request, error) {
	var (
		bodyReader io.Reader
		header     = p.Header
	)
	if p.Body != nil {
		bs, _ := json.Marshal(p.Body)
		bodyReader = bytes.NewReader(bs)
		header.Add(HeaderContentTypeKey, ContentTypeJSON)
	}

	if len(p.RawBody) > 0 {
		bodyReader = strings.NewReader(p.RawBody)
	}

	request, err := http.NewRequestWithContext(ctx, string(p.Method), p.getURL(), bodyReader)
	if err != nil {
		return nil, fmt.Errorf("new request failed, %w", err)
	}
	if header != nil {
		request.Header = header
	}
	return request, nil
}

func (p *HTTPParams) getClient() *http.Client {
	return &http.Client{
		Timeout: time.Duration(p.TimeoutSec * float64(time.Second)),
	}
}

func (p *HTTPParams) getResponseHandler() ResponseHandler {
	if p.RespHandler == "" {
		return DefaultResponseHandler
	}
	return p.RespHandler
}
