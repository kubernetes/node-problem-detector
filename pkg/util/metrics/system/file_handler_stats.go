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

package system

import (
	"fmt"
	"strconv"
	"strings"
)

const fileHandlerFilePath = "/proc/sys/fs/file-nr"

// GetFileHandleCurrentlyInUseCount returns the number of file handles currently
// in use.
func GetFileHandleCurrentlyInUseCount() (int64, error) {
	lines, err := ReadFileIntoLines(fileHandlerFilePath)
	if err != nil {
		return 0, fmt.Errorf("error retrieving the file handle count")
	}
	// fileHandlerFilePath contains only one line.
	// cat /proc/sys/fs/file-nr
	// 115123    13     814
	// line[0]: the total allocated file handles.
	// line[1]: the number of currently used file handles
	// line[2]: the maximum file handles that can be allocated
	splitLines := strings.Split(lines[0], "\t")

	return strconv.ParseInt(splitLines[1], 10, 64)

}
