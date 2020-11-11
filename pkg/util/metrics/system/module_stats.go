package kernel

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	)

type ModuleStat struct {
	ModuleName				string  `json:"moduleName"`
	Instances         uint64  `json:"instances"`
	Proprietary       bool    `json:"proprietary"`
	OutOfTree         bool    `json:"outOfTree"`
	Unsigned          bool    `json:"unsigned"`
}

// Module returns all the kernel modules and their
// usage. It is read from cat /proc/modules.
func Modules() ([]ModuleStat, error) {
	filename := "/proc/modules"
	if _, err := os.Stat(filename); err != nil {
		return nil, err
	}
	lines, _ := ReadFile(filename)
	var result = make([]ModuleStat, 0, len(lines))

	// a line of /proc/modules has the following structure
	// nf_nat 61440 2 xt_MASQUERADE,iptable_nat, Live 0x0000000000000000  (O)
	// (1)		(2)  (3)    (4)										 (5)	 	(6)								(7)
	for _, line := range lines {
		fields := strings.Fields(line)
		moduleName := fields[0]
		numberOfInstances, err := strconv.ParseUint((fields[1]), 10, 64)
		if err != nil {
			return nil, err
		}
		var isProprietary = false
		var isOutofTree = false
		var isUnsigned = false
		if len(fields) > 6 {
			isProprietary = strings.Contains(fields[6], "P")
			isOutofTree = strings.Contains(fields[6], "O")
			isUnsigned = strings.Contains(fields[6], "E")
		}
		var stats = ModuleStat{
			ModuleName:		moduleName,
			Instances: 		numberOfInstances,
			Proprietary:  isProprietary,
			OutOfTree:    isOutofTree,
			Unsigned: 		isUnsigned,
		}
		result = append(result, stats)
	}
	return result, nil
}

func ReadFile(filename string) ([]string, error) {
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