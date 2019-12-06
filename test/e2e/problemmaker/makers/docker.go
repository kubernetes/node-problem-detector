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

func init() {
	ProblemGenerators["DockerHung"] = makeDockerHung
}

func makeDockerHung() {
	const dockerHungPattern = `INFO: task docker:20744 blocked for more than 120 seconds.
      Tainted: G         C    3.16.0-4-amd64 #1
"echo 0 > /proc/sys/kernel/hung_task_timeout_secs" disables this message.
docker          D ffff8801a8f2b078     0 20744      1 0x00000000
 ffff8801a8f2ac20 0000000000000082 0000000000012f00 ffff880057a17fd8
 0000000000012f00 ffff8801a8f2ac20 ffffffff818bb4a0 ffff880057a17d80
 ffffffff818bb4a4 ffff8801a8f2ac20 00000000ffffffff ffffffff818bb4a8
Call Trace:
 [<ffffffff81510915>] ? schedule_preempt_disabled+0x25/0x70
 [<ffffffff815123c3>] ? __mutex_lock_slowpath+0xd3/0x1c0
 [<ffffffff815124cb>] ? mutex_lock+0x1b/0x2a
 [<ffffffff814175bc>] ? copy_net_ns+0x6c/0x130
 [<ffffffff8108bdf4>] ? create_new_namespaces+0xf4/0x180
 [<ffffffff8108beec>] ? copy_namespaces+0x6c/0x90
 [<ffffffff810654f6>] ? copy_process.part.25+0x966/0x1c30
 [<ffffffff81066991>] ? do_fork+0xe1/0x390
 [<ffffffff811c442c>] ? __alloc_fd+0x7c/0x120
 [<ffffffff81514079>] ? stub_clone+0x69/0x90
 [<ffffffff81513d0d>] ? system_call_fast_compare_end+0x10/0x15`

	writeKernelMessageOrDie(dockerHungPattern)
}
