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
)

func TestModules(t *testing.T) {
	modules, err := Modules()
	if err != nil {
		t.Errorf("error %v", err)
	}
	if modules == nil {
		t.Errorf("Could not get up time %v", v)
	}
}

func TestModuleStat_String(t *testing.T) {
	v := ModuleStat{
		ModuleName: "test",
		Instances:  2,
		OutOfTree:  false,
		Unsigned:   false,
	}
	e := `{"moduleName":"test", "instances": 2, Proprietary: false,"OutOfTree": false,"Unsigned":   false}`
	if e != fmt.Sprintf("%v", v) {
		t.Errorf("ModuleStat string is invalid:\ngot  %v\nexpected %v", v, e)
	}
}
