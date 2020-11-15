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
	"encoding/json"
	"fmt"
	"strings"
)

var cmdlineFilePath = "/proc/cmdline"

type CmdlineArg struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (d CmdlineArg) String() string {
	s, _ := json.Marshal(d)
	return string(s)
}

// As per the documentation kernel command line parameters can also be specified
// within double quotes and are separated with spaces.
// https://www.kernel.org/doc/html/v4.14/admin-guide/kernel-parameters.html#the-kernel-s-command-line-parameters
var withinQuotes = false

func splitAfterSpace(inputChar rune) bool {
	// words with space within double quotes cannot be split.
	// so we track the start quote and when we find the
	// end quote, then we reiterate.
	if inputChar == '"' {
		withinQuotes = !withinQuotes
		return false
	}
	//ignore spaces when it is within quotes.
	if withinQuotes {
		return false
	}
	return inputChar == ' '
}

// CmdlineArgs returns all the kernel cmdline. It is read from cat /proc/cmdline.
func CmdlineArgs() ([]CmdlineArg, error) {
	lines, err := ReadFileIntoLines(cmdlineFilePath)
	if err != nil {
		return nil, fmt.Errorf("error reading the file %v, %v", cmdlineFilePath, err)
	}
	if len(lines) < 1 {
		return nil, fmt.Errorf("no lines are retured")
	}
	cmdlineArgs := strings.FieldsFunc(lines[0], splitAfterSpace)
	var result = make([]CmdlineArg, 0, len(cmdlineArgs))
	// for commandline only one line is returned.
	for _, words := range cmdlineArgs {
		// Ignore the keys that start with double quotes
		if strings.Index(words, "\"") == 0 {
			continue
		}
		splitsWords := strings.Split(words, "=")
		if len(splitsWords) < 2 {
			var stats = CmdlineArg{
				Key: splitsWords[0],
			}
			result = append(result, stats)
		} else {
			//remove quotes in the values
			trimmedValue := strings.Trim(splitsWords[1], "\"'")
			var stats = CmdlineArg{
				Key:   splitsWords[0],
				Value: trimmedValue,
			}
			result = append(result, stats)
		}
	}
	return result, nil
}
