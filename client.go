package edgar

import (
	"crypto/tls"
	"fmt"
	"golang.org/x/time/rate"
	"io"
	"net/http"
)

type RequestError struct {
	StatusCode int
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("http request error: %d", e.StatusCode)
}

type Client struct {
	HttpClient *http.Client
	Limiter    *rate.Limiter
	UserAgent  string
}

func NewClient(userAgent string) *Client {
	return &Client{
		HttpClient: &http.Client{
			Transport: &http.Transport{TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)},
		},
		Limiter:   rate.NewLimiter(10, 10),
		UserAgent: userAgent,
	}
}

func (c *Client) Get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

func (c *Client) Post(url string, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", contentType)

	return c.Do(req)
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if err := c.Limiter.Wait(req.Context()); err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.UserAgent)

	return c.HttpClient.Do(req)
}
