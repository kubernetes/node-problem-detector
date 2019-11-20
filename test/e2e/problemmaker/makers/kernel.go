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
	"strings"

	"github.com/golang/glog"
)

func init() {
	ProblemGenerators["OOMKill"] = makeOOMKill
}

const kmsgPath = "/dev/kmsg"

func makeOOMKill() {
	const oomKillPattern = `Memory cgroup out of memory: Kill process 1012 (heapster) score 1035 or sacrifice child
Killed process 1012 (heapster) total-vm:327128kB, anon-rss:306328kB, file-rss:11132kB, shmem-rss:12345kB`

	writeKernelMessageOrDie(oomKillPattern)
}

func writeKernelMessageOrDie(msg string) {
	for _, line := range strings.Split(msg, "\n") {
		err := ioutil.WriteFile(kmsgPath, []byte(line), 0644)
		if err != nil {
			glog.Fatalf("Failed writting to %q: %v", kmsgPath, err)
		}
	}
}
