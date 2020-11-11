package system

import (
	"bufio"
	"os"
)

// ReadFile reads contents from a file and returns lines.
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
