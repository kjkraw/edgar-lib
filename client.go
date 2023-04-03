package edgar

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type RequestError struct {
	StatusCode int
}

func (e *RequestError) Error() string {
	return fmt.Sprintf("http request error: %d", e.StatusCode)
}

type Limiter struct {
	ticker    *time.Ticker
	close     chan bool
	cond      *sync.Cond
	limit     int
	remaining int
}

func NewLimiter(limit int) (l *Limiter) {
	l = &Limiter{
		ticker:    time.NewTicker(time.Second),
		cond:      sync.NewCond(&sync.Mutex{}),
		close:     make(chan bool),
		limit:     limit,
		remaining: limit,
	}
	go l.run()
	return l
}

func (l *Limiter) run() {
	defer l.ticker.Stop()
	for {
		select {
		case <-l.close:
			return
		case <-l.ticker.C:
			fmt.Printf("Tick!\n")
			l.cond.L.Lock()
			l.remaining = l.limit
			l.cond.Broadcast()
			l.cond.L.Unlock()
		}
	}
}

func (l *Limiter) Stop() {
	defer close(l.close)
	l.close <- true
}

func (l *Limiter) Wait() {
	l.cond.L.Lock()
	for l.remaining <= 0 {
		l.cond.Wait()
	}
	fmt.Printf("Remaining: %d\n", l.remaining)
	l.remaining--
	l.cond.L.Unlock()
}

type Client struct {
	HttpClient *http.Client
	Limiter    *Limiter
	UserAgent  string
}

func NewClient(userAgent string) *Client {
	return &Client{
		HttpClient: &http.Client{
			Transport: &http.Transport{TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)},
		},
		Limiter:   NewLimiter(10),
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
	c.Limiter.Wait()

	req.Header.Set("User-Agent", c.UserAgent)

	return c.HttpClient.Do(req)
}
