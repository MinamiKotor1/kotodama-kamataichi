package jsbox

import "syscall"

func setSandboxRlimits() error {
	const cpuSecs = 2
	_ = syscall.Setrlimit(syscall.RLIMIT_CPU, &syscall.Rlimit{Cur: cpuSecs, Max: cpuSecs})
	return nil
}
