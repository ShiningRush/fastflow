package http

import (
	"fmt"
	"github.com/shiningrush/fastflow/pkg/entity/run"
)

// HTTP action
type HTTP struct {
}

// Name Action name
func (h *HTTP) Name() string {
	return ActionHTTP
}

// ParameterNew
func (h *HTTP) ParameterNew() interface{} {
	return &HTTPParams{}
}

// Run
func (h *HTTP) Run(ctx run.ExecuteContext, params interface{}) error {
	p, ok := params.(*HTTPParams)
	if !ok {
		return fmt.Errorf("params type mismatch, want *HTTPParams, got %T", params)
	}
	err := p.validate()
	if err != nil {
		err = fmt.Errorf("validate HTTP Params failed, %w", err)
		ctx.Trace(err.Error())
		return err
	}

	request, err := p.buildRequest(ctx.Context())
	if err != nil {
		err = fmt.Errorf("build request failed, %w", err)
		ctx.Trace(err.Error())
		return err
	}
	cli := p.getClient()

	ctx.Tracef("start request %v", request.URL)
	response, err := cli.Do(request)
	if err != nil {
		err = fmt.Errorf("do http request failed, %w", err)
		ctx.Trace(err.Error())
		return err
	}

	httpResponse, err := ParseHTTPResponse(response, p)
	if err != nil {
		err = fmt.Errorf("parse http response failed, %w", err)
		ctx.Trace(err.Error())
		return err
	}

	key := p.RespSaveKey
	if key == "" {
		key = DefaultResponseSaveKey
	}
	ctx.WithValue(key, httpResponse)
	return nil
}
