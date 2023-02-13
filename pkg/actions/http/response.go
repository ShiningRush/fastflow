package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type HTTPResponse struct {
	ResponseHandler ResponseHandler
	Body            map[string]interface{}
	Raw             []byte
}

func ParseHTTPResponse(response *http.Response, p *HTTPParams) (*HTTPResponse, error) {
	if response.StatusCode/100 != 2 {
		bs, _ := ioutil.ReadAll(response.Body)
		return nil, fmt.Errorf("http failed: code:%d, message:%s, body:%s", response.StatusCode, response.Status, string(bs))
	}

	contentType := response.Header.Get(HeaderContentTypeKey)

	respBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body failed:%w", err)
	}

	body := map[string]interface{}{}
	resp := &HTTPResponse{
		ResponseHandler: p.getResponseHandler(),
		Body:            body,
		Raw:             respBytes,
	}

	switch p.getResponseHandler() {
	case ResponseHandlerNone:
		return resp, nil
	case ResponseHandlerJSON:
		decoder := json.NewDecoder(bytes.NewReader(respBytes))
		if p.UseNumber {
			decoder.UseNumber()
		}
		err := decoder.Decode(&body)
		return resp, err
		// TODO xml ... etc
	}
	return nil, fmt.Errorf("not supprt content type: %s, body: %s", contentType, string(respBytes))
}
