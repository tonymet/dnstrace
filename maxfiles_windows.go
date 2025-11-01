//go:build windows
// +build windows

package main

// Windows Compatibility for GetMaxOpenFiles
func GetMaxOpenFiles() (uint64, error) {
	// large number
	return 200000, nil // A very large, "safe" guess.
}
