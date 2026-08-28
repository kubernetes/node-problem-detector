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

package systemlogmonitor

import (
	"fmt"
	"regexp"
	"regexp/syntax"
	"slices"
	"strings"

	"k8s.io/node-problem-detector/pkg/systemlogmonitor/types"
)

// LogBuffer buffers the logs and matches a compiled pattern.
type LogBuffer interface {
	// Push pushes log into the log buffer.
	Push(*types.Log)
	// Match with a compiled pattern in the log buffer.
	Match(*Pattern) []*types.Log
}

// Pattern is a compiled rule plus the facts that let Match narrow its scan.
type Pattern struct {
	// regexp is the rule anchored to the end of the buffered logs.
	regexp *regexp.Regexp
	// lastLineOnly reports that the rule can match only in the last pushed line.
	// Match then skips building the joined buffer.
	lastLineOnly bool
}

// logBuffer is not safe for concurrent use.
type logBuffer struct {
	// buffer is a simple ring buffer.
	buffer  []*types.Log
	msg     []string
	max     int
	current int
	// joined caches the result of String. Push clears it.
	joined string
	// joinedOK reports whether joined is current.
	joinedOK bool
}

// NewLogBuffer creates log buffer with max line number limit. Because we only match logs
// in the log buffer, the max buffer line number is also the max pattern line number we
// support. Smaller buffer line number means less memory and cpu usage, but also means less
// lines of patterns we support.
func NewLogBuffer(maxLines int) *logBuffer {
	return &logBuffer{
		buffer: make([]*types.Log, maxLines),
		msg:    make([]string, maxLines),
		max:    maxLines,
	}
}

// CompilePattern compiles a log buffer pattern that must match to the end of
// the buffered logs.
func CompilePattern(expr string) (*Pattern, error) {
	// Compile expr alone first so an error cites the pattern as written.
	if _, err := regexp.Compile(expr); err != nil {
		return nil, err
	}
	// Group the expression so the end anchor applies to every top-level branch.
	anchored := `(?:` + expr + `)\z`
	reg, err := regexp.Compile(anchored)
	if err != nil {
		return nil, fmt.Errorf("invalid pattern %q: %w", expr, err)
	}
	p := &Pattern{regexp: reg}
	tree, err := syntax.Parse(anchored, syntax.Perl)
	if err != nil {
		return p, nil
	}
	p.lastLineOnly = isLastLineOnly(tree)
	return p, nil
}

// isLastLineOnly reports whether the rule accepts no newline and has no start anchor.
func isLastLineOnly(re *syntax.Regexp) bool {
	switch re.Op {
	case syntax.OpAnyChar:
		// `(?s).` accepts a newline.
		return false
	case syntax.OpBeginText, syntax.OpBeginLine, syntax.OpEndLine:
		// A start anchor marks the start of the whole buffer.
		return false
	case syntax.OpLiteral:
		if slices.Contains(re.Rune, '\n') {
			return false
		}
	case syntax.OpCharClass:
		// Rune stores the character class as inclusive lo, hi pairs.
		for i := 0; i+1 < len(re.Rune); i += 2 {
			if re.Rune[i] <= '\n' && '\n' <= re.Rune[i+1] {
				return false
			}
		}
	}
	for _, sub := range re.Sub {
		if !isLastLineOnly(sub) {
			return false
		}
	}
	return true
}

func (b *logBuffer) Push(log *types.Log) {
	b.buffer[b.current%b.max] = log
	b.msg[b.current%b.max] = log.Message
	b.current++
	b.joinedOK = false
	b.joined = ""
}

func (b *logBuffer) Match(p *Pattern) []*types.Log {
	if p.lastLineOnly {
		return b.matchLastLine(p.regexp)
	}
	log := b.String()
	loc := p.regexp.FindStringIndex(log)
	if loc == nil {
		// No match
		return nil
	}
	// reverse index
	s := len(log) - loc[0] - 1
	total := 0
	matched := []*types.Log{}
	for i := b.tail(); i >= b.current && b.buffer[i%b.max] != nil; i-- {
		matched = append(matched, b.buffer[i%b.max])
		total += len(b.msg[i%b.max]) + 1 // Add '\n'
		if total > s {
			break
		}
	}
	slices.Reverse(matched)
	return matched
}

// matchLastLine matches a lastLineOnly rule against the most recently pushed line.
func (b *logBuffer) matchLastLine(reg *regexp.Regexp) []*types.Log {
	if b.current == 0 {
		return nil
	}
	last := (b.current - 1) % b.max
	if !reg.MatchString(b.msg[last]) {
		return nil
	}
	return []*types.Log{b.buffer[last]}
}

func (b *logBuffer) String() string {
	if b.joinedOK {
		return b.joined
	}
	head := b.current % b.max
	lines := make([]string, 0, b.max)
	lines = append(lines, b.msg[head:]...)
	lines = append(lines, b.msg[:head]...)
	b.joined = concatLogs(lines)
	b.joinedOK = true
	return b.joined
}

// tail returns current tail index.
func (b *logBuffer) tail() int {
	return b.current + b.max - 1
}

// concatLogs concatenates multiple lines of logs into one string.
func concatLogs(logs []string) string {
	return strings.Join(logs, "\n")
}
