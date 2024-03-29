/*
Copyright 2023 The Kubernetes Authors All rights reserved.

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
	"io"

	"github.com/hpcloud/tail"
)

// getLogReader returns log reader for filelog log. Note that getLogReader doesn't look back
// to the rolled out logs.
func getLogReader(path string) (io.ReadCloser, error) {
	return tail.OpenFile(path)
}
