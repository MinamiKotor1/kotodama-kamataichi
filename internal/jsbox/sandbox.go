package jsbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"syscall"
	"time"

	"github.com/dop251/goja"
)

const (
	sandboxMaxInputBytes  = 1 * 1024 * 1024
	sandboxMaxCodeBytes   = 64 * 1024
	sandboxMaxOutputBytes = 256 * 1024
	sandboxTimeout        = 250 * time.Millisecond
)

type interruptSentinel struct{}

func RunSandbox() int {
	_ = setSandboxRlimits()
	runtime.GOMAXPROCS(1)

	in, err := io.ReadAll(io.LimitReader(os.Stdin, sandboxMaxInputBytes+1))
	if err != nil {
		writeResp(Response{OK: false, Error: err.Error()})
		return 0
	}
	if len(in) > sandboxMaxInputBytes {
		writeResp(Response{OK: false, Error: "request too large"})
		return 0
	}

	var req Request
	if err := json.Unmarshal(in, &req); err != nil {
		writeResp(Response{OK: false, Error: "invalid json"})
		return 0
	}
	if len(req.Code) == 0 {
		writeResp(Response{OK: false, Error: "missing code"})
		return 0
	}
	if len(req.Code) > sandboxMaxCodeBytes {
		writeResp(Response{OK: false, Error: "code too large"})
		return 0
	}
	if req.Kind != KindTransform {
		writeResp(Response{OK: false, Error: "unsupported kind"})
		return 0
	}

	ctx, cancel := context.WithTimeout(context.Background(), sandboxTimeout)
	defer cancel()

	out, err := runTransform(ctx, req.Code, req.Response)
	if err != nil {
		writeResp(Response{OK: false, Error: err.Error()})
		return 0
	}
	if len(out) > sandboxMaxOutputBytes {
		writeResp(Response{OK: false, Error: "result too large"})
		return 0
	}

	writeResp(Response{OK: true, Result: out})
	return 0
}

func writeResp(resp Response) {
	b, err := json.Marshal(resp)
	if err != nil {
		_, _ = io.WriteString(os.Stdout, `{"ok":false,"error":"internal"}`)
		return
	}
	_, _ = os.Stdout.Write(b)
}

func runTransform(ctx context.Context, src string, response any) (out json.RawMessage, err error) {
	rt := goja.New()
	_ = rt.Set("eval", goja.Undefined())
	_ = rt.Set("Function", goja.Undefined())

	done := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			rt.Interrupt(interruptSentinel{})
		case <-done:
		}
	}()
	defer close(done)

	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(interruptSentinel); ok {
				err = ctx.Err()
				return
			}
			panic(r)
		}
	}()

	v, err := rt.RunString("(" + src + ")")
	if err != nil {
		return nil, err
	}
	fn, ok := goja.AssertFunction(v)
	if !ok {
		return nil, errors.New("transform is not a function")
	}
	res, err := fn(goja.Undefined(), rt.ToValue(response))
	if err != nil {
		return nil, err
	}
	if goja.IsUndefined(res) {
		return nil, errors.New("transform returned undefined")
	}

	exported := res.Export()
	b, err := json.Marshal(exported)
	if err != nil {
		return nil, fmt.Errorf("transform result is not json: %w", err)
	}
	return b, nil
}

func setSandboxRlimits() error {
	// Best-effort resource limits. If the environment forbids it, we still rely on
	// the parent process timeout.
	const cpuSecs = 2
	_ = syscall.Setrlimit(syscall.RLIMIT_CPU, &syscall.Rlimit{Cur: cpuSecs, Max: cpuSecs})
	return nil
}
