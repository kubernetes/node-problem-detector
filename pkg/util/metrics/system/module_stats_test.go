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
	modules, err := Modules()
	if err != nil {
		t.Errorf("error %v", err)
	}
	if modules == nil {
		t.Error("Error retrieving modules")
	}
}

func TestModuleStat_String(t *testing.T) {
	v := ModuleStat{
		ModuleName: "test",
		Instances:  2,
		OutOfTree:  false,
		Unsigned:   false,
	}
	e := `{"moduleName":"test","instances":2,"proprietary":false,"outOfTree":false,"unsigned":false}`
	assert.Equal(t,
		e, fmt.Sprintf("%v", v), "ModuleStat string is invalid: %v", v)

}
