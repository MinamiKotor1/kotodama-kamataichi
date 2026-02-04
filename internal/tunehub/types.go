package tunehub

type APIResponse[T any] struct {
	Code    int    `json:"code"`
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

type MethodConfig struct {
	Type      string            `json:"type"`
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Params    map[string]any    `json:"params"`
	Body      map[string]any    `json:"body"`
	Headers   map[string]string `json:"headers"`
	Transform string            `json:"transform"`
}

type SearchItem struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Artist string `json:"artist"`
	Album  string `json:"album"`
}

type ParseRequest struct {
	Platform string `json:"platform"`
	IDs      string `json:"ids"`
	Quality  string `json:"quality"`
}

type ParseSongInfo struct {
	Name     string `json:"name"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Duration int    `json:"duration"`
}

type ParseItem struct {
	ID            string        `json:"id"`
	Success       bool          `json:"success"`
	URL           string        `json:"url"`
	Info          ParseSongInfo `json:"info"`
	Cover         string        `json:"cover"`
	Lyrics        string        `json:"lyrics"`
	Quality       string        `json:"quality"`
	ActualQuality string        `json:"actualQuality"`
	WasDowngraded bool          `json:"wasDowngraded"`
	FileSize      int64         `json:"fileSize"`
	Expire        int64         `json:"expire"`
	FromCache     bool          `json:"fromCache"`
	Error         string        `json:"error"`
}

type ParseData struct {
	Data          []ParseItem `json:"data"`
	Total         int         `json:"total"`
	SuccessCount  int         `json:"success_count"`
	FailCount     int         `json:"fail_count"`
	CacheHitCount int         `json:"cache_hit_count"`
	Cost          float64     `json:"cost"`
}
