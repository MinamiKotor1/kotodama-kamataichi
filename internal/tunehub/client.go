package tunehub

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"kotodama-kamataichi/internal/jsbox"
	"kotodama-kamataichi/internal/template"
)

const defaultBaseURL = "https://tunehub.sayqz.com/api"

type Client struct {
	BaseURL string
	HTTP    *http.Client
	HTTPv4  *http.Client
	JS      *jsbox.Runner
}

func New(js *jsbox.Runner) *Client {
	timeout := 25 * time.Second
	return &Client{
		BaseURL: defaultBaseURL,
		HTTP:    newHTTPClient(timeout, ""),
		HTTPv4:  newHTTPClient(timeout, "tcp4"),
		JS:      js,
	}
}

func newHTTPClient(timeout time.Duration, dialNetwork string) *http.Client {
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	tr := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if dialNetwork != "" {
				network = dialNetwork
			}
			return dialer.DialContext(ctx, network, addr)
		},
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: 15 * time.Second,
		IdleConnTimeout:       30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	if dialNetwork != "" {
		// Stick to HTTP/1.1 when forcing IPv4; avoids some HTTP/2 edge cases.
		tr.ForceAttemptHTTP2 = false
	}
	return &http.Client{Timeout: timeout, Transport: tr}
}

func (c *Client) GetMethods(ctx context.Context) (map[string][]string, error) {
	var resp APIResponse[map[string][]string]
	if err := c.getJSON(ctx, "/v1/methods", &resp); err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("tunehub: %s", resp.Message)
	}
	return resp.Data, nil
}

func (c *Client) GetMethodConfig(ctx context.Context, platform, function string) (MethodConfig, error) {
	platform = strings.TrimSpace(platform)
	function = strings.TrimSpace(function)
	if platform == "" || function == "" {
		return MethodConfig{}, errors.New("platform/function required")
	}
	var resp APIResponse[MethodConfig]
	if err := c.getJSON(ctx, "/v1/methods/"+url.PathEscape(platform)+"/"+url.PathEscape(function), &resp); err != nil {
		return MethodConfig{}, err
	}
	if resp.Code != 0 {
		return MethodConfig{}, fmt.Errorf("tunehub: %s", resp.Message)
	}
	return resp.Data, nil
}

func (c *Client) Search(ctx context.Context, platform, keyword string, page, limit int) ([]SearchItem, error) {
	if strings.EqualFold(strings.TrimSpace(platform), "qq") {
		return c.searchQQ(ctx, keyword, page, limit)
	}

	cfg, err := c.GetMethodConfig(ctx, platform, "search")
	if err != nil {
		return nil, err
	}
	vars := map[string]any{
		"keyword": keyword,
		"page":    page,
		"limit":   limit,
	}

	upstream, err := c.execMethod(ctx, cfg, vars)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.Transform) == "" {
		return nil, errors.New("missing transform")
	}
	if c.JS == nil {
		return nil, errors.New("jsbox runner not configured")
	}

	transformed, err := c.JS.Transform(ctx, cfg.Transform, upstream)
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(transformed)
	if err != nil {
		return nil, err
	}
	var items []SearchItem
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *Client) Parse(ctx context.Context, apiKey, platform, ids, quality string) (ParseData, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return ParseData{}, errors.New("missing api key")
	}
	reqBody := ParseRequest{Platform: platform, IDs: ids, Quality: quality}
	b, err := json.Marshal(reqBody)
	if err != nil {
		return ParseData{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/v1/parse", bytes.NewReader(b))
	if err != nil {
		return ParseData{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", apiKey)

	res, err := c.httpForHost(req.URL.Hostname()).Do(req)
	if err != nil {
		return ParseData{}, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 2*1024*1024))
	if err != nil {
		return ParseData{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return ParseData{}, fmt.Errorf("tunehub parse http %d", res.StatusCode)
	}

	var resp APIResponse[ParseData]
	if err := json.Unmarshal(body, &resp); err != nil {
		return ParseData{}, err
	}
	if resp.Code != 0 {
		return ParseData{}, fmt.Errorf("tunehub: %s", resp.Message)
	}
	return resp.Data, nil
}

func (c *Client) execMethod(ctx context.Context, cfg MethodConfig, vars map[string]any) (any, error) {
	if strings.ToLower(cfg.Type) != "http" {
		return nil, fmt.Errorf("unsupported method type: %q", cfg.Type)
	}
	method := strings.ToUpper(strings.TrimSpace(cfg.Method))
	if method != http.MethodGet && method != http.MethodPost {
		return nil, fmt.Errorf("unsupported http method: %q", cfg.Method)
	}
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, errors.New("missing url")
	}

	paramsAny, err := template.RenderAny(cfg.Params, vars)
	if err != nil {
		return nil, err
	}
	params, _ := paramsAny.(map[string]any)

	u, err := url.Parse(cfg.URL)
	if err != nil {
		return nil, err
	}
	rewriteNeteaseAPIHost(u)
	q := u.Query()
	for k, v := range params {
		q.Set(k, fmt.Sprint(v))
	}
	u.RawQuery = q.Encode()

	var bodyReader io.Reader
	if method == http.MethodPost && len(cfg.Body) > 0 {
		bodyAny, err := template.RenderAny(cfg.Body, vars)
		if err != nil {
			return nil, err
		}
		bb, err := json.Marshal(bodyAny)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(bb)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}
	for k, v := range cfg.Headers {
		req.Header.Set(k, v)
	}
	if method == http.MethodPost && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "kotodama-kamataichi")
	}
	res, err := c.httpForHost(req.URL.Hostname()).Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 4*1024*1024))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("upstream http %d", res.StatusCode)
	}

	var decoded any
	if err := json.Unmarshal(body, &decoded); err == nil {
		return decoded, nil
	}
	return string(body), nil
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	res, err := c.httpForHost(req.URL.Hostname()).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 2*1024*1024))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("tunehub http %d", res.StatusCode)
	}
	return json.Unmarshal(body, out)
}

func (c *Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Client) httpForHost(host string) *http.Client {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "music.163.com" || host == "interface.music.163.com" || host == "interface3.music.163.com" {
		if c.HTTPv4 != nil {
			return c.HTTPv4
		}
	}
	return c.http()
}

func rewriteNeteaseAPIHost(u *url.URL) {
	if u == nil {
		return
	}
	host := strings.TrimSpace(strings.ToLower(u.Hostname()))
	if host != "music.163.com" {
		return
	}
	if !strings.HasPrefix(u.Path, "/api/") {
		return
	}

	port := u.Port()
	newHost := "interface.music.163.com"
	if port != "" {
		newHost = net.JoinHostPort(newHost, port)
	}
	u.Host = newHost
}
