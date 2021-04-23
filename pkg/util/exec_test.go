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
	"runtime"
	"testing"
)

func TestExec(t *testing.T) {
	var cmds [][]string

	if runtime.GOOS == "windows" {
		cmds = [][]string{
			{"powershell.exe"},
			{"cmd.exe", "/C", "set", "/p", "$="},
			{"cmd.exe", "/K", "set", "/p", "$="},
			{"testdata/hello-world.cmd"},
			{"testdata/hello-world.bat"},
			{"testdata/hello-world.ps1"},
		}
	} else {
		cmds = [][]string{
			{"/bin/sh"},
			{"/bin/bash"},
		}
	}

	for _, v := range cmds {
		args := v
		t.Run(fmt.Sprintf("%v", args), func(t *testing.T) {
			cmd := Exec(args[0], args[1:]...)

			if err := Kill(cmd); err == nil {
				t.Error("Kill(cmd) expected to have error because of empty handle, got none")
			}

			if err := cmd.Start(); err != nil {
				t.Errorf("Start() got error, %v", err)
			}

			if err := Kill(cmd); err != nil {
				t.Errorf("Kill(cmd) for %s %v got error, %v", cmd.Path, cmd.Args, err)
			}
		})
	}
}
