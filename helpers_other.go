//go:build !windows

package main

import "os/exec"

func isAdmin() bool { return false }

func execOpen(path string) error {
	return exec.Command("xdg-open", path).Start()
}
