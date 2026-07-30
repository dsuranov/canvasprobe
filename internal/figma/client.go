package figma

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBaseURL      = "https://api.figma.com"
	defaultMaxBodyBytes = 100 * 1024 * 1024
)

type Client struct {
	httpClient *http.Client
	baseURL    *url.URL
	baseOrigin string
	token      string
	maxBody    int64
	userAgent  string
	verbose    bool
	stderr     io.Writer
	cache      *Cache
	jitterFn   func() time.Duration
	sleepFn    func(time.Duration)
}

type Options struct {
	Token        string
	BaseURL      string
	Timeout      time.Duration
	HTTPClient   *http.Client
	MaxBodyBytes int64
	UserAgent    string
	Verbose      bool
	Stderr       io.Writer
	Cache        *Cache
}

func New(opts Options) (*Client, error) {
	if opts.Token == "" {
		return nil, fmt.Errorf("no token: set CANVASPROBE_TOKEN, FIGMA_API_TOKEN, use --token-stdin, or configure token")
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid API base URL: %w", err)
	}
	if base.Scheme == "" || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("invalid API base URL")
	}
	if base.Path != "" && base.Path != "/" {
		return nil, fmt.Errorf("API base URL must not contain a path")
	}
	if baseURL != defaultBaseURL && opts.HTTPClient == nil {
		return nil, fmt.Errorf("custom API base URL requires an injected HTTP client")
	}
	if baseURL == defaultBaseURL && base.Scheme != "https" {
		return nil, fmt.Errorf("API base URL must use HTTPS")
	}

	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	maxBody := opts.MaxBodyBytes
	if maxBody == 0 {
		maxBody = defaultMaxBodyBytes
	}

	httpClient := &http.Client{}
	if opts.HTTPClient != nil {
		*httpClient = *opts.HTTPClient
	}
	if httpClient.Timeout == 0 {
		httpClient.Timeout = timeout
	}
	httpClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) == 0 {
			return nil
		}
		first := via[0].URL
		if req.URL.Scheme != first.Scheme || !strings.EqualFold(req.URL.Host, first.Host) {
			return fmt.Errorf("redirect blocked: destination origin differs")
		}
		if baseURL == defaultBaseURL && req.URL.Scheme != "https" {
			return fmt.Errorf("redirect blocked: HTTPS downgrade")
		}
		if len(via) >= 10 {
			return fmt.Errorf("redirect blocked: too many redirects")
		}
		return nil
	}

	userAgent := opts.UserAgent
	if userAgent == "" {
		userAgent = "canvasprobe/dev"
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    base,
		baseOrigin: base.Scheme + "://" + base.Host,
		token:      opts.Token,
		maxBody:    maxBody,
		userAgent:  userAgent,
		verbose:    opts.Verbose,
		stderr:     stderr,
		cache:      opts.Cache,
		jitterFn:   func() time.Duration { return time.Duration(rand.Int63n(500)) * time.Millisecond },
		sleepFn:    time.Sleep,
	}, nil
}

func (c *Client) GetMe(ctx context.Context) ([]byte, error) {
	return c.get(ctx, "/v1/me", nil)
}

func (c *Client) GetFile(ctx context.Context, fileKey string, depth int, ids []string, version string) ([]byte, error) {
	q := url.Values{}
	if depth > 0 {
		q.Set("depth", strconv.Itoa(depth))
	}
	if len(ids) > 0 {
		q.Set("ids", strings.Join(ids, ","))
	}
	if version != "" {
		q.Set("version", version)
	}
	return c.get(ctx, "/v1/files/"+url.PathEscape(fileKey), q)
}

func (c *Client) GetFileNodes(ctx context.Context, fileKey string, ids []string, depth int) ([]byte, error) {
	q := url.Values{}
	q.Set("ids", strings.Join(ids, ","))
	if depth > 0 {
		q.Set("depth", strconv.Itoa(depth))
	}
	return c.get(ctx, "/v1/files/"+url.PathEscape(fileKey)+"/nodes", q)
}

func (c *Client) GetFileComponents(ctx context.Context, fileKey string) ([]byte, error) {
	return c.get(ctx, "/v1/files/"+url.PathEscape(fileKey)+"/components", nil)
}

func (c *Client) GetFileStyles(ctx context.Context, fileKey string) ([]byte, error) {
	return c.get(ctx, "/v1/files/"+url.PathEscape(fileKey)+"/styles", nil)
}

func (c *Client) GetComments(ctx context.Context, fileKey string) ([]byte, error) {
	return c.get(ctx, "/v1/files/"+url.PathEscape(fileKey)+"/comments", nil)
}

func (c *Client) PostComment(ctx context.Context, fileKey, message, replyTo string) ([]byte, error) {
	body := struct {
		Message   string `json:"message"`
		CommentID string `json:"comment_id,omitempty"`
	}{
		Message:   message,
		CommentID: replyTo,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encode comment: %w", err)
	}
	return c.write(ctx, http.MethodPost, "/v1/files/"+url.PathEscape(fileKey)+"/comments", data)
}

func (c *Client) DeleteComment(ctx context.Context, fileKey, commentID string) error {
	_, err := c.write(ctx, http.MethodDelete, "/v1/files/"+url.PathEscape(fileKey)+"/comments/"+url.PathEscape(commentID), nil)
	return err
}

func (c *Client) get(ctx context.Context, path string, query url.Values) ([]byte, error) {
	return c.doGet(ctx, path, query, 1)
}

func (c *Client) doGet(ctx context.Context, path string, query url.Values, retriesLeft int) ([]byte, error) {
	queryStr := query.Encode()
	if c.cache != nil && retriesLeft == 1 {
		if data, ok := c.cache.Get(http.MethodGet, c.baseOrigin, path, queryStr, c.token); ok {
			return data, nil
		}
	}

	req, err := c.newRequest(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return nil, err
	}
	resp, elapsed, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	c.logRequest(http.MethodGet, path, resp.StatusCode, elapsed)

	if resp.StatusCode == http.StatusTooManyRequests && retriesLeft > 0 {
		_, _ = io.Copy(io.Discard, resp.Body)
		wait, _ := parseRetryAfter(resp.Header.Get("Retry-After"))
		if wait > 30*time.Second {
			return nil, ErrRateLimited
		}
		c.sleepFn(wait + c.jitterFn())
		return c.doGet(ctx, path, query, 0)
	}
	if resp.StatusCode >= 500 && retriesLeft > 0 {
		_, _ = io.Copy(io.Discard, resp.Body)
		c.sleepFn(time.Second + c.jitterFn())
		return c.doGet(ctx, path, query, 0)
	}

	data, err := c.processResponse(resp)
	if err != nil {
		return nil, err
	}
	if c.cache != nil {
		c.cache.Set(http.MethodGet, c.baseOrigin, path, queryStr, c.token, data)
	}
	return data, nil
}

func (c *Client) write(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	req, err := c.newRequest(ctx, method, path, nil, body)
	if err != nil {
		return nil, err
	}
	resp, elapsed, err := c.do(req)
	if err != nil {
		return nil, fmt.Errorf("%s result is unknown; request was not retried: %w", method, err)
	}
	defer resp.Body.Close()
	c.logRequest(method, path, resp.StatusCode, elapsed)
	return c.processResponse(resp)
}

func (c *Client) do(req *http.Request) (*http.Response, time.Duration, error) {
	start := time.Now()
	resp, err := c.httpClient.Do(req)
	return resp, time.Since(start), err
}

func (c *Client) logRequest(method, path string, status int, elapsed time.Duration) {
	if c.verbose {
		fmt.Fprintf(c.stderr, "[verbose] %s %s status=%d time=%s\n", method, path, status, elapsed.Round(time.Millisecond))
	}
}

func (c *Client) newRequest(ctx context.Context, method, path string, query url.Values, body []byte) (*http.Request, error) {
	if !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("API path must be absolute")
	}
	u := *c.baseURL
	u.Path = path
	u.RawPath = ""
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Figma-Token", c.token)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

func (c *Client) processResponse(resp *http.Response) ([]byte, error) {
	lr := io.LimitReader(resp.Body, c.maxBody+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > c.maxBody {
		return nil, ErrResponseTooLarge
	}

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return data, nil
	case resp.StatusCode == http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case resp.StatusCode == http.StatusForbidden:
		return nil, ErrForbidden
	case resp.StatusCode == http.StatusNotFound:
		return nil, ErrNotFound
	case resp.StatusCode == http.StatusTooManyRequests:
		return nil, ErrRateLimited
	default:
		body := string(data)
		if len(body) > 1024 {
			body = body[:1024]
		}
		msg := resp.Status
		var errBody struct {
			Err string `json:"err"`
		}
		if json.Unmarshal(data, &errBody) == nil && errBody.Err != "" {
			msg = errBody.Err
		}
		return nil, &APIError{
			Status:    resp.StatusCode,
			Message:   msg,
			Body:      body,
			RequestID: resp.Header.Get("X-Request-Id"),
		}
	}
}

func parseRetryAfter(header string) (time.Duration, bool) {
	s := strings.TrimSpace(header)
	if s == "" {
		return 0, false
	}
	if secs, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.Duration(secs) * time.Second, true
	}
	t, err := http.ParseTime(s)
	if err != nil {
		return 0, false
	}
	d := time.Until(t)
	if d < 0 {
		return 0, true
	}
	return d, true
}
