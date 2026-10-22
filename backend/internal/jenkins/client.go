package jenkins

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	userAgent   = "dbx-jenkins-plugin/0.4.0"
	maxJSONBody = 8 << 20
)

type crumbState struct {
	fetched  bool
	disabled bool
	field    string
	value    string
}

// Client 是 Jenkins REST 客户端。它持有 CookieJar、CSRF crumb 缓存和版本信息，
// 必须按连接复用；并发调用由内部锁保护。
type Client struct {
	baseURL    string
	username   string
	apiToken   string
	readOnly   bool
	httpClient *http.Client
	writeClient *http.Client

	mu      sync.Mutex
	version string
	crumb   crumbState
}

// Config 是连接 Jenkins 所需的全部配置。
type Config struct {
	BaseURL            string
	Username           string
	APIToken           string
	ReadOnly           bool
	InsecureSkipVerify bool
	Timeout            time.Duration
}

// Response 是写操作的轻量响应，避免把大响应体留在内存里。
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// WhoAmI 是连接测试的结果。
type WhoAmI struct {
	Name          string   `json:"name"`
	Authenticated bool     `json:"authenticated"`
	Authorities   []string `json:"authorities"`
	Mode          string   `json:"mode,omitempty"`
	Version       string   `json:"version,omitempty"`
}

// NewClient 校验配置并构造客户端（CookieJar / TLS / 超时）。
func NewClient(cfg Config) (*Client, error) {
	base := strings.TrimSpace(cfg.BaseURL)
	parsed, err := url.Parse(base)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, &APIError{Kind: ErrInvalidConfig, Message: "invalid Jenkins base URL"}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, &APIError{
			Kind:    ErrInvalidConfig,
			Message: fmt.Sprintf("unsupported scheme %q", parsed.Scheme),
		}
	}

	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, &APIError{Kind: ErrInvalidConfig, Message: "cannot create cookie jar"}
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{
		// 仅用于内网自签证书实例；默认关闭，需用户在连接表单显式开启。
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	httpClient := &http.Client{
		Transport: transport,
		Jar:       jar,
		Timeout:   timeout,
	}
	// Jenkins 的 doDelete / doRename / stop / cancelItem 成功时返回 302。
	// 写操作不跟随重定向，否则会被带到目标页面，把 404 误判成失败。
	writeClient := &http.Client{
		Transport: transport,
		Jar:       jar,
		Timeout:   timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &Client{
		baseURL:     strings.TrimRight(parsed.String(), "/"),
		username:    cfg.Username,
		apiToken:    cfg.APIToken,
		readOnly:    cfg.ReadOnly,
		httpClient:  httpClient,
		writeClient: writeClient,
	}, nil
}

// ReadOnly 返回连接是否处于只读模式。
func (c *Client) ReadOnly() bool { return c.readOnly }

// Version 返回最近一次响应头里的 Jenkins 版本。
func (c *Client) Version() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.version
}

// CloseIdleConnections 释放连接池里的空闲连接（断开连接时调用）。
func (c *Client) CloseIdleConnections() {
	c.httpClient.CloseIdleConnections()
}

// Get 发起 GET 请求并把 JSON 解码到 out。
func (c *Client) Get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.resolve(path), nil)
	if err != nil {
		return &APIError{Kind: ErrRequest, Message: err.Error()}
	}
	req.Header.Set("Accept", "application/json")

	response, err := c.do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(io.LimitReader(response.Body, maxJSONBody))
	if err != nil {
		return &APIError{Kind: ErrNetwork, Message: "cannot read Jenkins response"}
	}
	if response.StatusCode >= 400 {
		return newAPIError(response.StatusCode, data)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return &APIError{
			Kind:       ErrRequest,
			StatusCode: response.StatusCode,
			Message:    "cannot decode Jenkins response",
		}
	}
	return nil
}

// GetRaw 发起 GET 请求并返回原始响应体（用于 config.xml 等非 JSON 资源）。
func (c *Client) GetRaw(ctx context.Context, path, accept string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.resolve(path), nil)
	if err != nil {
		return nil, &APIError{Kind: ErrRequest, Message: err.Error()}
	}
	if strings.TrimSpace(accept) == "" {
		accept = "application/json"
	}
	request.Header.Set("Accept", accept)

	response, err := c.do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(io.LimitReader(response.Body, maxJSONBody))
	if err != nil {
		return nil, &APIError{Kind: ErrNetwork, Message: "cannot read Jenkins response"}
	}
	if response.StatusCode >= 400 {
		return nil, newAPIError(response.StatusCode, data)
	}
	return data, nil
}

// PostForm 发起表单 POST，自动处理 Basic Auth、Cookie、CSRF crumb 与一次重试。
func (c *Client) PostForm(ctx context.Context, path string, form url.Values) (*Response, error) {
	encoded := ""
	if form != nil {
		encoded = form.Encode()
	}
	return c.post(ctx, path, "application/x-www-form-urlencoded", []byte(encoded))
}

// PostBody 发起指定 Content-Type 的 POST（用于 config.xml 等 XML 写入）。
func (c *Client) PostBody(ctx context.Context, path, contentType string, body []byte) (*Response, error) {
	if strings.TrimSpace(contentType) == "" {
		contentType = "application/octet-stream"
	}
	return c.post(ctx, path, contentType, body)
}

func (c *Client) post(ctx context.Context, path, contentType string, body []byte) (*Response, error) {
	if c.readOnly {
		return nil, &APIError{Kind: ErrReadOnly, Message: "connection is in read-only mode"}
	}

	for attempt := 0; attempt < 2; attempt++ {
		crumb, err := c.ensureCrumb(ctx)
		if err != nil {
			return nil, err
		}

		request, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			c.resolve(path),
			bytes.NewReader(body),
		)
		if err != nil {
			return nil, &APIError{Kind: ErrRequest, Message: err.Error()}
		}
		request.Header.Set("Content-Type", contentType)
		if !crumb.disabled && crumb.field != "" {
			request.Header.Set(crumb.field, crumb.value)
		}

		response, err := c.doWith(request, c.writeClient)
		if err != nil {
			return nil, err
		}
		responseBody, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
		response.Body.Close()
		if readErr != nil {
			return nil, &APIError{Kind: ErrNetwork, Message: "cannot read Jenkins response"}
		}

		if response.StatusCode == http.StatusForbidden && attempt == 0 && isStaleCrumb(responseBody) {
			c.invalidateCrumb()
			continue
		}
		if response.StatusCode >= 400 {
			return nil, newAPIError(response.StatusCode, responseBody)
		}
		return &Response{
			StatusCode: response.StatusCode,
			Header:     response.Header.Clone(),
			Body:       responseBody,
		}, nil
	}

	return nil, &APIError{
		Kind:       ErrForbidden,
		StatusCode: http.StatusForbidden,
		Message:    "Jenkins rejected the CSRF crumb after refresh",
	}
}

// TestConnection 优先调用 /whoAmI/api/json，老版本回退 /api/json。
func (c *Client) TestConnection(ctx context.Context) (*WhoAmI, error) {
	var who WhoAmI
	err := c.Get(ctx, "/whoAmI/api/json", &who)
	if err == nil {
		who.Version = c.Version()
		return &who, nil
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) && apiErr.Kind == ErrNotFound {
		var legacy struct {
			Mode            string `json:"mode"`
			NodeDescription string `json:"nodeDescription"`
			UseSecurity     bool   `json:"useSecurity"`
		}
		if fallbackErr := c.Get(ctx, "/api/json?tree=mode,nodeDescription,useSecurity", &legacy); fallbackErr != nil {
			return nil, fallbackErr
		}
		return &WhoAmI{
			Name:          c.username,
			Authenticated: c.username != "",
			Mode:          legacy.Mode,
			Version:       c.Version(),
		}, nil
	}
	return nil, err
}

func (c *Client) resolve(path string) string {
	if path == "" {
		return c.baseURL
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return c.baseURL + path
}

func (c *Client) do(request *http.Request) (*http.Response, error) {
	return c.doWith(request, c.httpClient)
}

func (c *Client) doWith(request *http.Request, client *http.Client) (*http.Response, error) {
	request.Header.Set("User-Agent", userAgent)
	if request.Header.Get("Accept") == "" {
		request.Header.Set("Accept", "application/json")
	}
	if c.username != "" {
		request.SetBasicAuth(c.username, c.apiToken)
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, mapTransportError(err)
	}
	if version := response.Header.Get("X-Jenkins"); version != "" {
		c.mu.Lock()
		c.version = version
		c.mu.Unlock()
	}
	return response, nil
}

func (c *Client) ensureCrumb(ctx context.Context) (crumbState, error) {
	c.mu.Lock()
	if c.crumb.fetched {
		state := c.crumb
		c.mu.Unlock()
		return state, nil
	}
	c.mu.Unlock()

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.resolve("/crumbIssuer/api/json"),
		nil,
	)
	if err != nil {
		return crumbState{}, &APIError{Kind: ErrRequest, Message: err.Error()}
	}
	response, err := c.do(request)
	if err != nil {
		return crumbState{}, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
		var payload struct {
			Crumb string `json:"crumb"`
			Field string `json:"crumbRequestField"`
		}
		if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil ||
			payload.Crumb == "" || payload.Field == "" {
			return c.storeCrumb(crumbState{fetched: true, disabled: true}), nil
		}
		return c.storeCrumb(crumbState{
			fetched: true,
			field:   payload.Field,
			value:   payload.Crumb,
		}), nil
	case http.StatusNotFound, http.StatusForbidden:
		// CSRF 保护未启用，或当前账号无权读取 crumb；按无 crumb 继续。
		return c.storeCrumb(crumbState{fetched: true, disabled: true}), nil
	default:
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return crumbState{}, newAPIError(response.StatusCode, body)
	}
}

func (c *Client) storeCrumb(state crumbState) crumbState {
	c.mu.Lock()
	c.crumb = state
	c.mu.Unlock()
	return state
}

func (c *Client) invalidateCrumb() {
	c.mu.Lock()
	c.crumb = crumbState{}
	c.mu.Unlock()
}

func isStaleCrumb(body []byte) bool {
	text := strings.ToLower(string(body))
	return strings.Contains(text, "no valid crumb") ||
		(strings.Contains(text, "crumb") && strings.Contains(text, "invalid"))
}

func newAPIError(status int, body []byte) *APIError {
	kind := ErrRequest
	switch {
	case status == http.StatusUnauthorized:
		kind = ErrUnauthorized
	case status == http.StatusForbidden:
		kind = ErrForbidden
	case status == http.StatusNotFound:
		kind = ErrNotFound
	case status >= 500:
		kind = ErrServer
	}

	message := strings.TrimSpace(string(body))
	switch {
	case message == "":
		message = http.StatusText(status)
	case looksLikeHTML(message):
		// Jenkins 的 500 常返回整页 HTML，原样透传对用户没有意义。
		message = fmt.Sprintf(
			"Jenkins returned an HTML error page (HTTP %d); check the Jenkins server log",
			status,
		)
	}
	if len(message) > 300 {
		message = message[:300]
	}
	return &APIError{Kind: kind, StatusCode: status, Message: message}
}

func looksLikeHTML(text string) bool {
	lower := strings.ToLower(text)
	return strings.HasPrefix(lower, "<!doctype") || strings.HasPrefix(lower, "<html")
}

func mapTransportError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return &APIError{Kind: ErrTimeout, Message: "Jenkins request timed out"}
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return &APIError{Kind: ErrTimeout, Message: "Jenkins request timed out"}
	}
	return &APIError{Kind: ErrNetwork, Message: "cannot reach Jenkins: " + err.Error()}
}
