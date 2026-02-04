package download

import (
	"strings"
	"unicode"
)

func sanitizeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "unknown"
	}

	var b strings.Builder
	b.Grow(len(s))
	lastUnderscore := false
	for _, r := range s {
		switch {
		case r == ' ' || r == '\t':
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		case r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|':
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		case unicode.IsControl(r):
			if !lastUnderscore {
				b.WriteByte('_')
				lastUnderscore = true
			}
		default:
			b.WriteRune(r)
			lastUnderscore = false
		}
	}

	out := strings.Trim(b.String(), "._ ")
	if out == "" {
		return "unknown"
	}
	if len(out) > 120 {
		out = out[:120]
	}
	return out
}
