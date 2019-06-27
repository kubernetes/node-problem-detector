// osrelease is a go package to make reading the contents of os-release files easier
//
// See https://www.freedesktop.org/software/systemd/man/os-release.html
package osrelease

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"strings"
)

const EtcOsRelease string = "/etc/os-release"
const UsrLibOsRelease string = "/usr/lib/os-release"

// Read and return os-release, trying EtcOsRelease, followed by UsrLibOsRelease.
// err will contain an error message if neither file exists or failed to parse
func Read() (osrelease map[string]string, err error) {
	osrelease, err = ReadFile(EtcOsRelease)
	if err != nil {
		osrelease, err = ReadFile(UsrLibOsRelease)
	}
	return
}

// Similar to Read(), but takes the name of a file to load instead
func ReadFile(filename string) (osrelease map[string]string, err error) {
	osrelease = make(map[string]string)
	err = nil

	lines, err := parseFile(filename)
	if err != nil {
		return
	}

	for _, v := range lines {
		key, value, err := parseLine(v)
		if err == nil {
			osrelease[key] = value
		}
	}
	return
}

// ReadString is similar to Read(), but takes a string to load instead
func ReadString(content string) (osrelease map[string]string, err error) {
	osrelease = make(map[string]string)
	err = nil

	lines, err := parseString(content)
	if err != nil {
		return
	}

	for _, v := range lines {
		key, value, err := parseLine(v)
		if err == nil {
			osrelease[key] = value
		}
	}
	return
}

func parseFile(filename string) (lines []string, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

func parseString(content string) (lines []string, err error) {
	in := bytes.NewBufferString(content)
	reader := bufio.NewReader(in)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()

}

func parseLine(line string) (key string, value string, err error) {
	err = nil

	// skip empty lines
	if len(line) == 0 {
		err = errors.New("Skipping: zero-length")
		return
	}

	// skip comments
	if line[0] == '#' {
		err = errors.New("Skipping: comment")
		return
	}

	// try to split string at the first '='
	splitString := strings.SplitN(line, "=", 2)
	if len(splitString) != 2 {
		err = errors.New("Can not extract key=value")
		return
	}

	// trim white space from key and value
	key = splitString[0]
	key = strings.Trim(key, " ")
	value = splitString[1]
	value = strings.Trim(value, " ")

	// Handle double quotes
	if strings.ContainsAny(value, `"`) {
		first := string(value[0:1])
		last := string(value[len(value)-1:])

		if first == last && strings.ContainsAny(first, `"'`) {
			value = strings.TrimPrefix(value, `'`)
			value = strings.TrimPrefix(value, `"`)
			value = strings.TrimSuffix(value, `'`)
			value = strings.TrimSuffix(value, `"`)
		}
	}

	// expand anything else that could be escaped
	value = strings.Replace(value, `\"`, `"`, -1)
	value = strings.Replace(value, `\$`, `$`, -1)
	value = strings.Replace(value, `\\`, `\`, -1)
	value = strings.Replace(value, "\\`", "`", -1)
	return
}
