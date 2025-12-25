//go:build !windows

package main

import (
	"os/exec"
)

func winHiddenCMDFrom(cmd *exec.Cmd) {
	// 非windows系统 所以不需要这个参数来隐藏黑窗口
	_ = cmd
}
