// Package elife is the library behind the elife command: the HTTP client,
// request shaping, and the typed data models for eLife Sciences.
//
// The eLife API is fully public (no auth required) and returns HAL+JSON with
// versioned vendor media types. Setting the correct Accept header is required
// to avoid 406 Not Acceptable responses.
// Base URL: https://api.elifesciences.org
package elife

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DefaultUserAgent identifies the client to the eLife API.
const DefaultUserAgent = "elife/dev (+https://github.com/tamnd/elife-cli)"

// ErrNotFound is returned when the API returns 404 for an article or resource.
var ErrNotFound = errors.New("not found")

// Accept headers required by the eLife HAL+JSON API.
const (
	acceptArticleList = "application/vnd.elife.article-list+json; version=1"
	acceptArticle     = "application/vnd.elife.article-vor+json; version=8, application/vnd.elife.article-poa+json; version=4"
	acceptSubjectList = "application/vnd.elife.subject-list+json; version=1"
)

// Config holds constructor parameters for NewClient.
type Config struct {
	BaseURL   string
	UserAgent string
	Rate      time.Duration
	Retries   int
	Timeout   time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		BaseURL:   "https://api.elifesciences.org",
		UserAgent: DefaultUserAgent,
		Rate:      300 * time.Millisecond,
		Retries:   3,
		Timeout:   15 * time.Second,
	}
}

// Client talks to the eLife Sciences API.
type Client struct {
	cfg        Config
	httpClient *http.Client
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client configured with cfg.
func NewClient(cfg Config) *Client {
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.elifesciences.org"
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = DefaultUserAgent
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 15 * time.Second
	}
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

// get fetches a URL with the given Accept header, pacing and retrying.
func (c *Client) get(ctx context.Context, rawURL, accept string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		body, retry, err := c.do(ctx, rawURL, accept)
		if err == nil {
			return body, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL, accept string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, false, ErrNotFound
	}
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

// getJSON fetches and JSON-decodes into v.
func (c *Client) getJSON(ctx context.Context, rawURL, accept string, v any) error {
	body, err := c.get(ctx, rawURL, accept)
	if err != nil {
		return err
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "null" {
		return ErrNotFound
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode %s: %w", rawURL, err)
	}
	return nil
}

// ─── API methods ──────────────────────────────────────────────────────────────

// Recent returns articles ordered by publication date.
func (c *Client) Recent(ctx context.Context, opts ListOptions) ([]ArticleItem, int, error) {
	if opts.PerPage <= 0 {
		opts.PerPage = 20
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.Order == "" {
		opts.Order = "desc"
	}

	params := url.Values{}
	params.Set("per-page", strconv.Itoa(opts.PerPage))
	params.Set("page", strconv.Itoa(opts.Page))
	params.Set("order", opts.Order)
	if opts.Subject != "" {
		params.Set("subject[]", opts.Subject)
	}
	if opts.Type != "" {
		params.Set("type[]", opts.Type)
	}

	rawURL := c.cfg.BaseURL + "/articles?" + params.Encode()
	var resp articleListResp
	if err := c.getJSON(ctx, rawURL, acceptArticleList, &resp); err != nil {
		return nil, 0, err
	}

	items := make([]ArticleItem, 0, len(resp.Items))
	for i, w := range resp.Items {
		items = append(items, wireToArticleItem(w, i+1))
	}
	return items, resp.Total, nil
}

// Search performs a full-text search and returns matching articles.
func (c *Client) Search(ctx context.Context, opts SearchOptions) ([]ArticleItem, int, error) {
	if opts.PerPage <= 0 {
		opts.PerPage = 20
	}
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.Order == "" {
		opts.Order = "desc"
	}

	params := url.Values{}
	params.Set("for", opts.Query)
	params.Set("per-page", strconv.Itoa(opts.PerPage))
	params.Set("page", strconv.Itoa(opts.Page))
	params.Set("order", opts.Order)

	rawURL := c.cfg.BaseURL + "/search?" + params.Encode()
	var resp articleListResp
	if err := c.getJSON(ctx, rawURL, acceptArticleList, &resp); err != nil {
		return nil, 0, err
	}

	items := make([]ArticleItem, 0, len(resp.Items))
	for i, w := range resp.Items {
		items = append(items, wireToArticleItem(w, i+1))
	}
	return items, resp.Total, nil
}

// Article fetches a single article by its eLife ID.
func (c *Client) Article(ctx context.Context, id string) (Article, error) {
	rawURL := c.cfg.BaseURL + "/articles/" + url.PathEscape(id)
	var w articleDetailWire
	if err := c.getJSON(ctx, rawURL, acceptArticle, &w); err != nil {
		return Article{}, fmt.Errorf("article %q: %w", id, err)
	}
	return wireToArticle(w), nil
}

// Subjects returns the list of subject areas.
func (c *Client) Subjects(ctx context.Context) ([]Subject, error) {
	params := url.Values{}
	params.Set("per-page", "100")
	rawURL := c.cfg.BaseURL + "/subjects?" + params.Encode()
	var resp subjectListResp
	if err := c.getJSON(ctx, rawURL, acceptSubjectList, &resp); err != nil {
		return nil, err
	}
	subjects := make([]Subject, 0, len(resp.Items))
	for i, s := range resp.Items {
		subjects = append(subjects, Subject{
			Rank: i + 1,
			ID:   s.ID,
			Name: s.Name,
		})
	}
	return subjects, nil
}
