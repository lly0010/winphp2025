//go:build !windows

package wincmd

import "os/exec"

func hide(cmd *exec.Cmd) {}
