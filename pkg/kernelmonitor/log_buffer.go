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

package kernelmonitor

import (
	"regexp"
	"strings"

	"k8s.io/node-problem-detector/pkg/kernelmonitor/types"
)

// LogBuffer buffers the logs and supports match in the log buffer with regular expression.
type LogBuffer interface {
	// Push pushes log into the log buffer.
	Push(*types.KernelLog)
	// Match with regular expression in the log buffer.
	Match(string) []*types.KernelLog
	// String returns a concatenated string of the buffered logs.
	String() string
}

type logBuffer struct {
	// buffer is a simple ring buffer.
	buffer  []*types.KernelLog
	msg     []string
	max     int
	current int
}

// NewLogBuffer creates log buffer with max line number limit. Because we only match logs
// in the log buffer, the max buffer line number is also the max pattern line number we
// support. Smaller buffer line number means less memory and cpu usage, but also means less
// lines of patterns we support.
func NewLogBuffer(maxLines int) *logBuffer {
	return &logBuffer{
		buffer: make([]*types.KernelLog, maxLines, maxLines),
		msg:    make([]string, maxLines, maxLines),
		max:    maxLines,
	}
}

func (b *logBuffer) Push(log *types.KernelLog) {
	b.buffer[b.current%b.max] = log
	b.msg[b.current%b.max] = log.Message
	b.current++
}

// TODO(random-liu): Cache regexp if garbage collection becomes a problem someday.
func (b *logBuffer) Match(expr string) []*types.KernelLog {
	// The expression should be checked outside, and it must match to the end.
	reg := regexp.MustCompile(expr + `\z`)
	log := b.String()
	loc := reg.FindStringIndex(log)
	if loc == nil {
		// No match
		return nil
	}
	// reverse index
	s := len(log) - loc[0] - 1
	total := 0
	matched := []*types.KernelLog{}
	for i := b.tail(); i >= b.current && b.buffer[i%b.max] != nil; i-- {
		matched = append(matched, b.buffer[i%b.max])
		total += len(b.msg[i%b.max]) + 1 // Add '\n'
		if total > s {
			break
		}
	}
	for i := 0; i < len(matched)/2; i++ {
		matched[i], matched[len(matched)-i-1] = matched[len(matched)-i-1], matched[i]
	}
	return matched
}

func (b *logBuffer) String() string {
	logs := append(b.msg[b.current%b.max:], b.msg[:b.current%b.max]...)
	return concatLogs(logs)
}

// tail returns current tail index.
func (b *logBuffer) tail() int {
	return b.current + b.max - 1
}

// concatLogs concatenates multiple lines of logs into one string.
func concatLogs(logs []string) string {
	return strings.Join(logs, "\n")
}
