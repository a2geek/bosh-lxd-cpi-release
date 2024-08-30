package throttle

import (
	"bosh-lxd-cpi/config"
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	transactionsRoot = "http://localhost/transactions"
)

func NewThrottleClient(config config.ThrottleConfig) (ThrottleClient, error) {
	conn, err := net.Dial("unix", config.Path)
	if err != nil {
		return ThrottleClient{}, err
	}

	return ThrottleClient{
		client: http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, network, addr string) (net.Conn, error) {
					return conn, nil
				},
			},
		},
	}, nil
}

type ThrottleClient struct {
	client http.Client
}

func (c *ThrottleClient) Lock() (string, int, error) {
	r, err := c.client.Post(transactionsRoot, "plain/text", strings.NewReader(""))
	statusCode := 500
	if r != nil {
		statusCode = r.StatusCode
	}
	if err != nil {
		return "", statusCode, err
	}
	defer r.Body.Close()
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return "", statusCode, err
	}
	return string(data), statusCode, nil
}

// LockAndWait will handle the "too many requests" (429) response until it is free.
func (c *ThrottleClient) LockAndWait() (string, error) {
	var transactionId string
	for transactionId == "" {
		content, statusCode, err := c.Lock()
		if statusCode == http.StatusTooManyRequests {
			time.Sleep(10 * time.Second)
			statusCode = 0
			err = nil
		}
		if err != nil {
			return "", err
		}
		transactionId = content
	}
	return transactionId, nil
}

func (c *ThrottleClient) Unlock(transactionId string) error {
	url, err := url.JoinPath(transactionsRoot, transactionId)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
