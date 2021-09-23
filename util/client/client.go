package client

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/jakubDoka/keeper/util"
	"github.com/jakubDoka/keeper/util/uuid"
)

type Client struct {
	link   string
	secret string
	http.Client
}

func New(scheme, host, secret string, port int) (*Client, error) {
	client := &Client{
		link:   fmt.Sprintf("%s://%s:%d/", scheme, host, port),
		secret: secret,
	}

	err := client.Ping()
	if err != nil {
		return nil, util.WrapErr("failed to ping server: ", err)
	}

	return client, nil
}

func (c *Client) Rpc(id, meta, format string, session uuid.UUID, data []byte) (*http.Response, error) {
	var req http.Request

	req.Method = "POST"

	if format == "" {
		format = "application/json"
	}

	req.Header.Add("Content-Type", format)
	req.Header.Add("meta", meta)
	req.Header.Add("session", session.String())
	req.Header.Add("id", id)

	req.Body = io.NopCloser(bytes.NewReader(data))

	resp, err := c.Do(&req)
	if err != nil {
		return nil, util.WrapErr("failed to send request: ", err)
	}

	return resp, nil
}

func (c *Client) Ping() error {
	resp, err := c.Get(c.link)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s", resp.Status)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return util.WrapErr("failed to read body: ", err)
	}

	if string(bytes) != c.secret {
		return fmt.Errorf("server responded with different secret: %s", string(bytes))
	}

	return nil
}
