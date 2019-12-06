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

package makers

import (
	"io/ioutil"

	"github.com/golang/glog"
)

func init() {
	ProblemGenerators["Ext4FilesystemError"] = makeFilesystemError
}

const ext4ErrorTrigger = "/sys/fs/ext4/sda1/trigger_fs_error"

func makeFilesystemError() {
	msg := []byte("fake filesystem error from problem-maker")
	err := ioutil.WriteFile(ext4ErrorTrigger, msg, 0200)
	if err != nil {
		glog.Fatalf("Failed writting log to %q: %v", ext4ErrorTrigger, err)
	}
}
