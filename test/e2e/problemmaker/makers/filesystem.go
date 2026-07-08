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

const ext4ErrorPattern = `EXT4-fs error (device sda1): fake filesystem error from problem-maker
EXT4-fs (sda1): Remounting filesystem read-only`

func init() {
	ProblemGenerators["Ext4FilesystemError"] = makeFilesystemError
}

func makeFilesystemError() {
	writeKernelMessageOrDie(ext4ErrorPattern)
}
