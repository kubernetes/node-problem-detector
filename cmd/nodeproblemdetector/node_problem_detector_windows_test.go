//go:build !disable_system_log_monitor
// +build !disable_system_log_monitor

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

package main

import (
	"testing"

	"golang.org/x/sys/windows/svc"
)

func TestWindowsServiceLoop(t *testing.T) {
	npdo, cleanup := setupNPD(t)
	defer cleanup()

	setupLogging(false)

	s := &npdService{
		options: npdo,
	}

	r := make(chan svc.ChangeRequest, 2)
	changes := make(chan svc.Status, 4)
	defer func() {
		close(r)
		close(changes)
	}()

	r <- svc.ChangeRequest{
		Cmd: svc.Shutdown,
	}
	r <- svc.ChangeRequest{
		Cmd: svc.Shutdown,
	}

	ssec, errno := s.Execute([]string{}, r, changes)
	if ssec != false {
		t.Error("ssec should be false")
	}
	if errno != 0 {
		t.Error("errno should be 0")
	}
}
