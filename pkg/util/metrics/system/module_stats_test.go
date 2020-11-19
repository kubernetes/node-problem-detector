/*
Copyright 2019 The Kubernetes Authors All rights reserved.
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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModules(t *testing.T) {
	testcases := []struct {
		name               string
		fakeModuleFilePath string
		expectedModules    []Module
	}{
		{
			name:               "default_cos",
			fakeModuleFilePath: "testdata/modules_cos.txt",
			expectedModules: []Module{
				{
					ModuleName:  "crypto_simd",
					Instances:   0x1,
					Proprietary: false,
					OutOfTree:   false,
					Unsigned:    false,
				},
				{
					ModuleName:  "virtio_balloon",
					Instances:   0x0,
					Proprietary: false,
					OutOfTree:   false,
					Unsigned:    false,
				},
				{
					ModuleName:  "cryptd",
					Instances:   0x1,
					Proprietary: false,
					OutOfTree:   false,
					Unsigned:    false,
				},
				{
					ModuleName:  "loadpin_trigger",
					Instances:   0x0,
					Proprietary: false,
					OutOfTree:   true,
					Unsigned:    false,
				},
			},
		},
		{
			name:               "default_ubuntu",
			fakeModuleFilePath: "testdata/modules_ubuntu.txt",
			expectedModules: []Module{
				{
					ModuleName:  "drm",
					Instances:   0x0,
					Proprietary: false,
					OutOfTree:   false,
					Unsigned:    false,
				},
				{
					ModuleName:  "virtio_rng",
					Instances:   0x0,
					Proprietary: false,
					OutOfTree:   false,
					Unsigned:    false,
				},
				{
					ModuleName:  "x_tables",
					Instances:   0x1,
					Proprietary: false,
					OutOfTree:   false,
					Unsigned:    false,
				},
				{
					ModuleName:  "autofs4",
					Instances:   0x2,
					Proprietary: false,
					OutOfTree:   false,
					Unsigned:    false,
				},
			},
		},
	}
	for _, test := range testcases {
		t.Run(test.name, func(t *testing.T) {
			originalModuleFilePath := modulesFilePath
			defer func() {
				modulesFilePath = originalModuleFilePath
			}()

			modulesFilePath = test.fakeModuleFilePath
			modules, err := Modules()
			if err != nil {
				t.Errorf("Unexpected error retrieving modules: %v\nModulesFilePath: %s\n", err, modulesFilePath)
			}
			assert.Equal(t, modules, test.expectedModules, "unpected modules retrieved: %v, expected: %v", modules, test.expectedModules)
		})
	}
}

func TestModuleStat_String(t *testing.T) {
	v := Module{
		ModuleName: "test",
		Instances:  2,
		OutOfTree:  false,
		Unsigned:   false,
	}
	e := `{"moduleName":"test","instances":2,"proprietary":false,"outOfTree":false,"unsigned":false}`
	assert.Equal(t,
		e, fmt.Sprintf("%v", v), "Module string is invalid: %v", v)

}
