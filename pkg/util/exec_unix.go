//go:build unix

/*
Copyright 2021 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"fmt"
	"os/exec"
	"syscall"
)

// Exec creates a new process with the specified arguments.
func Exec(name string, arg ...string) *exec.Cmd {
	// create a process group
	sysProcAttr := &syscall.SysProcAttr{
		Setpgid: true,
	}
	cmd := exec.Command(name, arg...)
	cmd.SysProcAttr = sysProcAttr
	return cmd
}

// Kill the process and subprocesses.
func Kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return fmt.Errorf("%v does not have a process handle", cmd)
	}
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
