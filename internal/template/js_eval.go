package template

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/dop251/goja"
)

const evalTimeout = 50 * time.Millisecond

type interruptSentinel struct{}

var (
	identRE       = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)
	allowedExprRE = regexp.MustCompile(`^[A-Za-z0-9_$\s().|+\-*/%<>=!&?:,]+$`)
	forbiddenRe   = regexp.MustCompile(`(?i)\b(function|return|new|this|while|for|do|switch|case|try|catch|class|import|export|throw|delete|var|let|const)\b`)
)

func EvalExpr(expr string, vars map[string]any) (any, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, errors.New("empty expression")
	}
	if len(expr) > 512 {
		return nil, errors.New("expression too long")
	}
	if strings.Contains(expr, "{") || strings.Contains(expr, "}") || strings.Contains(expr, "[") || strings.Contains(expr, "]") {
		return nil, errors.New("invalid characters")
	}
	if strings.Contains(expr, "\"") || strings.Contains(expr, "'") || strings.Contains(expr, "`") {
		return nil, errors.New("quotes not allowed")
	}
	if strings.Contains(expr, ";") || strings.Contains(expr, "\\") || strings.Contains(expr, "\n") || strings.Contains(expr, "\r") {
		return nil, errors.New("invalid characters")
	}
	if strings.Contains(expr, "/*") || strings.Contains(expr, "*/") || strings.Contains(expr, "//") {
		return nil, errors.New("comments not allowed")
	}
	if !allowedExprRE.MatchString(expr) {
		return nil, errors.New("expression contains forbidden characters")
	}
	if forbiddenRe.MatchString(expr) {
		return nil, errors.New("expression contains forbidden keywords")
	}

	ctx, cancel := context.WithTimeout(context.Background(), evalTimeout)
	defer cancel()

	rt := goja.New()
	_ = rt.Set("eval", goja.Undefined())
	_ = rt.Set("Function", goja.Undefined())
	for k, v := range vars {
		if !identRE.MatchString(k) {
			return nil, fmt.Errorf("invalid var name: %q", k)
		}
		if err := rt.Set(k, v); err != nil {
			return nil, err
		}
	}

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			rt.Interrupt(interruptSentinel{})
		case <-done:
		}
	}()
	defer close(done)

	var interrupted any
	defer func() {
		if r := recover(); r != nil {
			interrupted = r
		}
	}()

	v, err := rt.RunString(`"use strict"; (` + expr + `)`)
	if interrupted != nil {
		if _, ok := interrupted.(interruptSentinel); ok {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("js panic: %v", interrupted)
	}
	if err != nil {
		return nil, err
	}
	if goja.IsUndefined(v) {
		return nil, errors.New("undefined")
	}
	return v.Export(), nil
}
