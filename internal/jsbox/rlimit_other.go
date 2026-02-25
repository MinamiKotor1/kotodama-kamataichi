//go:build !linux

package jsbox

func setSandboxRlimits() error { return nil }
