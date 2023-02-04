// Copyright 2016 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tail

import (
	"io"
	"testing"
)

func TestReadNewTail(t *testing.T) {
	// Read should return (0, io.EOF) before first
	// attemptOpen.
	tail, err := newTail("test/nonexist/file")
	if err != nil {
		t.Fatalf("Failed to create tail: %v", err)
	}
	buf := make([]byte, 0, 100)
	n, err := tail.Read(buf)
	if n != 0 {
		t.Errorf("Expected 0 bytes read, got %d", n)
	}
	if len(buf) != 0 {
		t.Errorf("Expected 0 bytes in buffer, got %d", len(buf))
	}
	if err != io.EOF {
		t.Errorf("Expected EOF, got %v", err)
	}
}
