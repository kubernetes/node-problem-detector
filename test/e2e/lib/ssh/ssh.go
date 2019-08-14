/*
Copyright 2019 The Kubernetes Authors.

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

package ssh

import (
	"fmt"
	"net"
	"os"

	k8s_ssh "k8s.io/kubernetes/pkg/ssh"
)

// Result holds the execution result of SSH command.
type Result struct {
	User     string
	Host     string
	Cmd      string
	Stdout   string
	Stderr   string
	Code     int
	SSHError error
}

// Run synchronously SSHs to a machine and runs cmd.
func Run(cmd, ip, sshUser, sshKey string) Result {
	result := Result{User: sshUser, Host: ip, Cmd: cmd}

	if result.User == "" {
		result.User = os.Getenv("USER")
	}

	if ip == "" {
		result.SSHError = fmt.Errorf("empty IP address")
		return result
	}

	signer, err := k8s_ssh.MakePrivateKeySignerFromFile(sshKey)
	if err != nil {
		result.SSHError = fmt.Errorf("error getting signer from key file %s: %v", sshKey, err)
		return result
	}

	stdout, stderr, code, err := k8s_ssh.RunSSHCommand(cmd, result.User, net.JoinHostPort(ip, "22"), signer)
	result.Stdout = stdout
	result.Stderr = stderr
	result.Code = code
	result.SSHError = err

	return result
}
