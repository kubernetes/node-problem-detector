/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

package filelog

import (
	"fmt"
	"io"
	"os"

	"github.com/google/cadvisor/utils/tail"
)

// getLogReader returns log reader for filelog log. Note that getLogReader doesn't look back
// to the rolled out logs.
func getLogReader(path string) (io.ReadCloser, error) {
	if path == "" {
		return nil, fmt.Errorf("unexpected empty log path")
	}
	// To handle log rotation, tail will not report error immediately if
	// the file doesn't exist. So we check file existence first.
	// This could go wrong during mid-rotation. It should recover after
	// several restart when the log file is created again. The chance
	// is slim but we should still fix this in the future.
	// TODO(random-liu): Handle log missing during rotation.
	_, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat the file %q: %v", path, err)
	}
	tail, err := tail.NewTail(path)
	if err != nil {
		return nil, fmt.Errorf("failed to tail the file %q: %v", path, err)
	}
	return tail, nil
}
