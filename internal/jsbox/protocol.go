package jsbox

import "encoding/json"

type Kind string

const (
	KindTransform Kind = "transform"
)

type Request struct {
	Kind     Kind   `json:"kind"`
	Code     string `json:"code"`
	Response any    `json:"response,omitempty"`
}

type Response struct {
	OK     bool            `json:"ok"`
	Error  string          `json:"error,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
}
