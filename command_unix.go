// +build !windows

package test

import (
	"os/exec"
	"runtime"
)

func command(instance Instance) *exec.Cmd {
	if runtime.GOOS == "darwin" {
		return exec.Command("open", instance.ID)
	}

	return exec.Command("xdg-open", instance.ID)
}
