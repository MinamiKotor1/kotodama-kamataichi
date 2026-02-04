package template

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var placeholderRe = regexp.MustCompile(`\{\{\s*([^}]+?)\s*\}\}`)

func RenderAny(v any, vars map[string]any) (any, error) {
	switch x := v.(type) {
	case string:
		return RenderString(x, vars)
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, vv := range x {
			r, err := RenderAny(vv, vars)
			if err != nil {
				return nil, err
			}
			out[k] = r
		}
		return out, nil
	case []any:
		out := make([]any, 0, len(x))
		for _, vv := range x {
			r, err := RenderAny(vv, vars)
			if err != nil {
				return nil, err
			}
			out = append(out, r)
		}
		return out, nil
	default:
		return v, nil
	}
}

func RenderString(s string, vars map[string]any) (string, error) {
	matches := placeholderRe.FindAllStringSubmatchIndex(s, -1)
	if len(matches) == 0 {
		return s, nil
	}

	var b strings.Builder
	last := 0
	for _, m := range matches {
		if len(m) != 4 {
			return "", errors.New("invalid placeholder match")
		}
		start, end := m[0], m[1]
		exprStart, exprEnd := m[2], m[3]
		b.WriteString(s[last:start])

		expr := strings.TrimSpace(s[exprStart:exprEnd])
		val, err := EvalExpr(expr, vars)
		if err != nil {
			return "", fmt.Errorf("eval %q: %w", expr, err)
		}
		b.WriteString(formatValue(val))
		last = end
	}
	b.WriteString(s[last:])
	return b.String(), nil
}

func formatValue(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "true"
		}
		return "false"
	case float64:
		if x == float64(int64(x)) {
			return strconv.FormatInt(int64(x), 10)
		}
		return strconv.FormatFloat(x, 'g', -1, 64)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	default:
		return fmt.Sprint(v)
	}
}
