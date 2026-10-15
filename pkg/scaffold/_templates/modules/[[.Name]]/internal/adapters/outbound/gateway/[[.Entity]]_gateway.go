// Package gateway holds the HTTP clients to the external providers this
// module sends through. They implement the ports; the provider's URL and
// token come in through the constructor so the adapter never reads config.
package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/rotisserie/eris"

	"[[.OrgPrefix]]/[[.Name]]/internal/domain/[[.Entity]]"
)

const requestTimeout = 10 * time.Second

// HTTP[[.Entity | pascal]]Gateway implements ports.[[.Entity | pascal]]Gateway over
// the provider's HTTP API.
type HTTP[[.Entity | pascal]]Gateway struct {
	client *http.Client
	url    string
	token  string
}

// NewHTTP[[.Entity | pascal]]Gateway creates the gateway for the given endpoint.
func NewHTTP[[.Entity | pascal]]Gateway(url, token string) *HTTP[[.Entity | pascal]]Gateway {
	return &HTTP[[.Entity | pascal]]Gateway{client: &http.Client{Timeout: requestTimeout}, url: url, token: token}
}

// [[.Entity | pascal]]Send POSTs the message as JSON. Any transport error or
// non-2xx status is [[.Entity]].ErrGatewayUnavailable.
// ponytail: the body is a placeholder; shape it to the provider's API.
func (g *HTTP[[.Entity | pascal]]Gateway) [[.Entity | pascal]]Send(ctx context.Context, a *[[.Entity]].[[.Entity | pascal]]) error {
	payload, err := json.Marshal(map[string]string{"label": a.Label().String()})
	if err != nil {
		return eris.Wrap(err, "encoding [[.Entity]]")
	}

	res, err := g.do(ctx, http.MethodPost, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode > 299 {
		return eris.Wrapf([[.Entity]].ErrGatewayUnavailable, "provider answered %d", res.StatusCode)
	}

	return nil
}

// Health GETs the endpoint and returns the status code; a 5xx answer is
// [[.Entity]].ErrGatewayUnavailable.
func (g *HTTP[[.Entity | pascal]]Gateway) Health(ctx context.Context) (any, error) {
	res, err := g.do(ctx, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= http.StatusInternalServerError {
		return nil, eris.Wrapf([[.Entity]].ErrGatewayUnavailable, "provider answered %d", res.StatusCode)
	}

	return res.StatusCode, nil
}

func (g *HTTP[[.Entity | pascal]]Gateway) do(ctx context.Context, method string, payload *bytes.Reader) (*http.Response, error) {
	var in io.Reader = http.NoBody
	if payload != nil {
		in = payload
	}
	httpReq, err := http.NewRequestWithContext(ctx, method, g.url, in)
	if err != nil {
		return nil, eris.Wrap(err, "building provider request")
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if g.token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+g.token)
	}

	res, err := g.client.Do(httpReq)
	if err != nil {
		return nil, eris.Wrapf([[.Entity]].ErrGatewayUnavailable, "calling provider: %v", err)
	}

	return res, nil
}
