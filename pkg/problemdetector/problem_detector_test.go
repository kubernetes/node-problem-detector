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

package problemdetector

import (
	"testing"

	"k8s.io/node-problem-detector/pkg/types"
)

func TestEmpty(t *testing.T) {
	pd := NewProblemDetector(map[string]types.Monitor{}, []types.Exporter{})
	if err := pd.Run(nil, nil); err == nil {
		t.Error("expected error when running an empty problem detector")
	}
}
