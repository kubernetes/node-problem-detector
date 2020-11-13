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
	"strconv"
	"strings"
)

type ModuleStat struct {
	ModuleName  string `json:"moduleName"`
	Instances   uint64 `json:"instances"`
	Proprietary bool   `json:"proprietary"`
	OutOfTree   bool   `json:"outOfTree"`
	Unsigned    bool   `json:"unsigned"`
}

func (d ModuleStat) String() string {
	s, _ := json.Marshal(d)
	return string(s)
}

// Module returns all the kernel modules and their
// usage. It is read from cat /proc/modules.
func Modules() ([]ModuleStat, error) {
	filename := "/proc/modules"
	lines, err := ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Error reading the contents of %s: %s", filename, err)
	}
	var result = make([]ModuleStat, 0, len(lines))

	/* a line of /proc/modules has the following structure
	  nf_nat 61440 2 xt_MASQUERADE,iptable_nat, Live 0x0000000000000000  (O)
	  (1)		(2)  (3)    (4)										 (5)	 	(6)								(7)
		(1)  name of the module
		(2)	 memory size of the module, in bytes
		(3)  instances of the module are currently loaded
		(4)  module dependencies
		(5)  load state of the module: live, loading or unloading
		(6)  memory offset for the loaded module.
		(7)  return a string to represent the kernel taint state. (used here: "P" - Proprietary, "O" - out of tree kernel module, "E" - unsigned module
	*/
	for _, line := range lines {
		fields := strings.Fields(line)
		moduleName := fields[0] // name of the module
		numberOfInstances, err :=
			strconv.ParseUint((fields[1]), 10, 64) // instances of the module are currently loaded
		if err != nil {
			return nil, err
		}

		var isProprietary = false
		var isOutofTree = false
		var isUnsigned = false
		// if the len of the fields is greater than 6, then the kernel taint state is available.
		if len(fields) > 6 {
			isProprietary = strings.Contains(fields[6], "P")
			isOutofTree = strings.Contains(fields[6], "O")
			isUnsigned = strings.Contains(fields[6], "E")
		}
		var stats = ModuleStat{
			ModuleName:  moduleName,
			Instances:   numberOfInstances,
			Proprietary: isProprietary,
			OutOfTree:   isOutofTree,
			Unsigned:    isUnsigned,
		}
		result = append(result, stats)
	}
	return result, nil
}
