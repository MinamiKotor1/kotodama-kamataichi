package jsbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

const (
	defaultTimeout        = 300 * time.Millisecond
	defaultMaxStderrBytes = 8 * 1024
	defaultMaxStdoutBytes = 256 * 1024
)

type Runner struct {
	ExecPath       string
	Timeout        time.Duration
	MaxStdoutBytes int64
	MaxStderrBytes int64
}

func NewRunner() (*Runner, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return &Runner{
		ExecPath:       exe,
		Timeout:        defaultTimeout,
		MaxStdoutBytes: defaultMaxStdoutBytes,
		MaxStderrBytes: defaultMaxStderrBytes,
	}, nil
}

func (r *Runner) Transform(ctx context.Context, code string, response any) (any, error) {
	if r == nil {
		return nil, errors.New("jsbox runner is nil")
	}
	req := Request{Kind: KindTransform, Code: code, Response: response}
	var out any
	if err := r.call(ctx, req, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *Runner) call(ctx context.Context, req Request, out any) error {
	if r.ExecPath == "" {
		return errors.New("jsbox ExecPath is empty")
	}
	if r.Timeout <= 0 {
		r.Timeout = defaultTimeout
	}
	if r.MaxStdoutBytes <= 0 {
		r.MaxStdoutBytes = defaultMaxStdoutBytes
	}
	if r.MaxStderrBytes <= 0 {
		r.MaxStderrBytes = defaultMaxStderrBytes
	}

	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	input, err := json.Marshal(req)
	if err != nil {
		return err
	}

	cmd := exec.CommandContext(ctx, r.ExecPath, "js-sandbox")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		_, _ = stdin.Write(input)
		_ = stdin.Close()
	}()

	readPipe := func(r io.Reader, maxBytes int64) ([]byte, error) {
		b, err := io.ReadAll(io.LimitReader(r, maxBytes))
		if err != nil {
			return nil, err
		}
		// Drain remaining data so the child can't block on a full pipe.
		_, _ = io.Copy(io.Discard, r)
		return b, nil
	}

	var (
		stdoutBytes []byte
		stderrBytes []byte
		stdoutErr   error
		stderrErr   error
	)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		stdoutBytes, stdoutErr = readPipe(stdout, r.MaxStdoutBytes+1)
	}()
	go func() {
		defer wg.Done()
		stderrBytes, stderrErr = readPipe(stderr, r.MaxStderrBytes)
	}()

	waitErr := cmd.Wait()
	wg.Wait()

	if stdoutErr != nil {
		return stdoutErr
	}
	if stderrErr != nil {
		// stderr read errors are rare; surface them for visibility.
		return stderrErr
	}
	if int64(len(stdoutBytes)) > r.MaxStdoutBytes {
		return fmt.Errorf("js-sandbox stdout exceeded limit (%d bytes)", r.MaxStdoutBytes)
	}

	if waitErr != nil {
		errText := string(bytes.TrimSpace(stderrBytes))
		if errText == "" {
			errText = waitErr.Error()
		}
		return fmt.Errorf("js-sandbox failed: %s", errText)
	}

	var resp Response
	if err := json.Unmarshal(stdoutBytes, &resp); err != nil {
		errText := string(bytes.TrimSpace(stderrBytes))
		if errText != "" {
			return fmt.Errorf("js-sandbox bad json: %w (stderr=%s)", err, errText)
		}
		return fmt.Errorf("js-sandbox bad json: %w", err)
	}
	if !resp.OK {
		if resp.Error == "" {
			resp.Error = "unknown js-sandbox error"
		}
		return errors.New(resp.Error)
	}
	if out == nil {
		return nil
	}
	if len(resp.Result) == 0 {
		return errors.New("js-sandbox returned empty result")
	}
	if err := json.Unmarshal(resp.Result, out); err != nil {
		return err
	}
	return nil
}
