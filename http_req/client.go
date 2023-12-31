package http_req

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	zerologloki "github.com/thisismz/zerolog-loki"
)

const (
	readTimeOut        = 10
	maxIdleConnections = 128
	maxConnections     = 512
	path               = ""
)

type HttpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type promHttpClient struct {
	username   string
	password   string
	hostUrl    string
	httpClient HttpDoer
}

func New(host, username, password string, opts ...Option) (*promHttpClient, error) {
	pc := &promHttpClient{
		hostUrl:  host,
		username: username,
		password: password,
	}

	for _, opt := range opts {
		err := opt.applyOption(pc)
		if err != nil {
			return nil, err
		}
	}

	return pc, nil
}

func (p *promHttpClient) Push(ctx context.Context, req zerologloki.PushRequest) error {

	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, p.hostUrl+path, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return err
	}

	request.Header.Add("content-type", "application/json")
	if p.password != "" {
		request.SetBasicAuth(p.username, p.password)
	}
	// set timeout
	request = request.WithContext(ctx)

	resp, err := p.httpClient.Do(request)
	if err != nil {
		return err
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("error: %s", string(respBody))
	}

	return nil
}
