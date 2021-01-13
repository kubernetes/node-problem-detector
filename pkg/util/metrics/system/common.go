/*
Copyright 2020 The Kubernetes Authors All rights reserved.
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
	"bufio"
	"os"
)

// ReadFileIntoLines reads contents from a file and returns lines.
func ReadFileIntoLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []string
	s := bufio.NewScanner(file)
	for s.Scan() {
		result = append(result, s.Text())
	}
	if s.Err() != nil {
		return nil, err
	}
	return result, nil
}

func ContainsModule(key string, values []Module) bool {
	for _, val := range values {
		if val.ModuleName == key {
			return true
		}
	}
	return false
}
