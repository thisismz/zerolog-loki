package client

import (
	"context"
	"net/http"
	"time"

	zerologloki "github.com/thisismz/zerolog-loki"
	promtailHttpClient "github.com/thisismz/zerolog-loki/http_req"
)

const (
	readTimeOut        = 100
	writeTimeOut       = 100
	maxIdleConnections = 128
	maxConnections     = 512
)

func newHttpClient() *http.Client {
	customTransport := &(*http.DefaultTransport.(*http.Transport))

	customTransport.MaxConnsPerHost = maxConnections
	customTransport.MaxIdleConnsPerHost = maxIdleConnections
	customTransport.MaxIdleConns = maxIdleConnections

	return &http.Client{
		Transport: customTransport,
		Timeout:   readTimeOut * time.Second,
	}
}

type promClient struct {
	httpClient   zerologloki.HttpClient
	streamConv   zerologloki.StreamConverter
	staticLabels map[string]interface{}
	writeTimeout time.Duration
}

func NewSimpleClient(host, username, password string, opts ...Option) (*promClient, error) {

	pHttpClient, err := promtailHttpClient.New(host, username, password,
		promtailHttpClient.WithHttpClient(newHttpClient()),
	)

	if err != nil {
		return nil, err
	}
	client := &promClient{
		streamConv: zerologloki.NewRawStreamConv("", ""),
		httpClient: pHttpClient,
	}

	for _, opt := range opts {
		if err = opt.applyOption(client); err != nil {
			return nil, err
		}
	}

	return client, nil
}

func (c *promClient) Write(p []byte) (i int, err error) {

	ctx, cancel := context.WithTimeout(context.Background(), c.writeTimeout)
	defer cancel()

	labels, err := c.streamConv.ExtractLabels(p)
	if err != nil {
		return 0, err
	}

	entry, err := c.streamConv.ConvertEntry(p)
	if err != nil {
		return 0, err
	}

	for k, v := range c.staticLabels {
		labels[k] = v
	}

	req := zerologloki.PushRequest{Streams: []*zerologloki.Stream{{
		Labels:  labels,
		Entries: []zerologloki.Entry{entry},
	}}}

	if rErr := c.httpClient.Push(ctx, req); rErr != nil {
		return 0, rErr
	}

	return len(p), nil
}
