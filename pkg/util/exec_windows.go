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
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// Exec creates a new process with the specified arguments.
func Exec(name string, arg ...string) *exec.Cmd {
	// Windows does not handle relative path names in exec very well.
	name = filepath.Clean(name)
	cmdArgs := arg

	// Detect scripts via file extension and automatically invoke them within a shell.
	// This mirrors the Linux behavior if the execute bit on file where a shell context
	// is automatically created.
	switch strings.ToLower(filepath.Ext(name)) {
	// Batch Scripts
	case ".cmd", ".bat":
		cmdArgs = append([]string{"/C", name}, cmdArgs...)
		name = "cmd.exe"
	// Powershell Scripts
	case ".ps1":
		cmdArgs = append([]string{name}, cmdArgs...)
		return Powershell(cmdArgs...)
	default:
		// Run directly.
	}

	return exec.Command(name, cmdArgs...)
}

// Powershell creates a new powershell process with the specified arguments
func Powershell(args ...string) *exec.Cmd {
	defaultFlags := []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "RemoteSigned"}
	args = append(defaultFlags, args...)
	return exec.Command("powershell.exe", args...)
}

// ExitStatus returns the exit code of the application.
func ExitStatus(cmd *exec.Cmd) int {
	return cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()
}

// Kill the process and subprocesses.
func Kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return fmt.Errorf("%v does not have a process handle", cmd)
	}

	// Use taskkill to kill the child process by process id.
	// /F = Force
	// /T = Kill child processes.
	// https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/taskkill
	kill := exec.Command("TASKKILL", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
	kill.Stderr = os.Stderr
	kill.Stdout = os.Stdout
	err := kill.Run()
	if execErr, ok := err.(*exec.ExitError); ok {
		// Error code 128 (ERROR_WAIT_NO_CHILDREN) means that taskkill couldn't find the process, it probably died already.
		// https://docs.microsoft.com/en-us/windows/win32/debug/system-error-codes--0-499-
		if execErr.ExitCode() == 128 {
			return nil
		}
	}
	return err
}
