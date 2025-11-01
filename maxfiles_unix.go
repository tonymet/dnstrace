//go:build unix
// +build unix

package main

import (
	"fmt"
	"syscall"
)

// GetMaxOpenFiles returns the process's soft limit for file descriptors on Unix-like systems.
func GetMaxOpenFiles() (uint64, error) {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return 0, fmt.Errorf("error getting RLIMIT_NOFILE: %w", err)
	}
	// We return the soft limit (Cur) as this is the enforced limit.
	return rLimit.Cur, nil
}
