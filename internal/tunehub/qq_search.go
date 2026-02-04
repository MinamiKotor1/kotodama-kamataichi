package tunehub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

func (c *Client) searchQQ(ctx context.Context, keyword string, page, limit int) ([]SearchItem, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}

	u, _ := url.Parse("https://c.y.qq.com/soso/fcgi-bin/client_search_cp")
	q := u.Query()
	q.Set("w", keyword)
	q.Set("p", fmt.Sprint(page))
	q.Set("n", fmt.Sprint(limit))
	q.Set("format", "json")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Referer", "https://y.qq.com/")
	req.Header.Set("User-Agent", "kotodama-kamataichi")

	res, err := c.http().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("qq upstream http %d", res.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, 2*1024*1024))
	if err != nil {
		return nil, err
	}

	var payload struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Song struct {
				List []struct {
					SongMid   string `json:"songmid"`
					SongName  string `json:"songname"`
					AlbumName string `json:"albumname"`
					Singer    []struct {
						Name string `json:"name"`
					} `json:"singer"`
				} `json:"list"`
			} `json:"song"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	if payload.Code != 0 {
		msg := strings.TrimSpace(payload.Message)
		if msg == "" {
			msg = "qq search failed"
		}
		return nil, fmt.Errorf("qq upstream: %s", msg)
	}

	items := make([]SearchItem, 0, len(payload.Data.Song.List))
	for _, s := range payload.Data.Song.List {
		artists := make([]string, 0, len(s.Singer))
		for _, a := range s.Singer {
			name := strings.TrimSpace(a.Name)
			if name != "" {
				artists = append(artists, name)
			}
		}
		items = append(items, SearchItem{
			ID:     strings.TrimSpace(s.SongMid),
			Name:   strings.TrimSpace(s.SongName),
			Artist: strings.Join(artists, ", "),
			Album:  strings.TrimSpace(s.AlbumName),
		})
	}
	return items, nil
}
