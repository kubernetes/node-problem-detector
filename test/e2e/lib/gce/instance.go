/*
Copyright 2018 The Kubernetes Authors.

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

package gce

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/golang/glog"
	. "github.com/onsi/gomega"
	compute "google.golang.org/api/compute/v1"
)

// Instance represents a GCE instance.
type Instance struct {
	Name           string
	Zone           string
	Project        string
	MachineType    string
	ExternalIP     string
	SshKey         string
	SshUser        string
	ComputeService *compute.Service
}

// CreateInstance creates a GCE instance with provided spec.
func CreateInstance(instance Instance, imageName string, imageProject string) (Instance, error) {
	if instance.MachineType == "" {
		instance.MachineType = "n1-standard-1"
	}

	p, err := instance.ComputeService.Projects.Get(instance.Project).Do()
	if err != nil {
		return instance, fmt.Errorf("failed to get project info %q: %v", instance.Project, err)
	}

	i := &compute.Instance{
		Name:        instance.Name,
		MachineType: fmt.Sprintf("zones/%s/machineTypes/%s", instance.Zone, instance.MachineType),
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				AccessConfigs: []*compute.AccessConfig{
					{
						Type: "ONE_TO_ONE_NAT",
						Name: "External NAT",
					},
				}},
		},
		Disks: []*compute.AttachedDisk{
			{
				AutoDelete: true,
				Boot:       true,
				Type:       "PERSISTENT",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					SourceImage: fmt.Sprintf("projects/%s/global/images/%s", imageProject, imageName),
					DiskSizeGb:  20,
				},
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: p.DefaultServiceAccount,
				Scopes: []string{
					"https://www.googleapis.com/auth/cloud-platform",
				},
			},
		},
	}

	if _, err := instance.ComputeService.Instances.Get(instance.Project, instance.Zone, instance.Name).Do(); err != nil {
		op, err := instance.ComputeService.Instances.Insert(instance.Project, instance.Zone, i).Do()
		if err != nil {
			ret := fmt.Sprintf("could not create instance %s: API error: %v", instance.Name, err)
			if op != nil {
				ret = fmt.Sprintf("%s: %v", ret, op.Error)
			}
			return instance, fmt.Errorf(ret)
		} else if op.Error != nil {
			return instance, fmt.Errorf("could not create instance %s: %+v", instance.Name, op.Error)
		}
	}

	instanceRunning := false
	// Waiting for the instance to be SSH-able by retrying SSH with timeout of 5 min (15*20sec)
	for i := 0; i < 15 && !instanceRunning; i++ {
		if i > 0 {
			time.Sleep(time.Second * 20)
		}
		if instance.ExternalIP == "" {
			instance.populateExternalIP()
		}

		_, err := instance.RunCommand("pwd")
		if err != nil {
			err = fmt.Errorf("SSH to instance %s failed: %s", instance.Name, err)
			continue
		}
		instanceRunning = true
	}

	if !instanceRunning {
		return instance, err
	}
	return instance, nil
}

func (ins *Instance) populateExternalIP() {
	gceInstance, err := ins.ComputeService.Instances.Get(ins.Project, ins.Zone, ins.Name).Do()
	if err != nil {
		return
	}

	for i := range gceInstance.NetworkInterfaces {
		ni := gceInstance.NetworkInterfaces[i]
		for j := range ni.AccessConfigs {
			ac := ni.AccessConfigs[j]
			if len(ac.NatIP) > 0 {
				ins.ExternalIP = ac.NatIP
				return
			}
		}
	}
}

// RunCommand runs a command on the GCE instance and returns the command result.
func (ins *Instance) RunCommand(cmd string) (string, error) {
	if ins.ExternalIP == "" {
		ins.populateExternalIP()
	}

	cmdLine := []string{"ssh",
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-i", ins.SshKey,
		"-p", "22",
		fmt.Sprintf("%s@%s", ins.SshUser, ins.ExternalIP),
		cmd}
	output, err := runCommand(cmdLine)
	return output, err
}

// RunCommand runs a command on the GCE instance and returns the command result,
// and fails the test when the command failed.
func (ins *Instance) RunCommandOrFail(cmd string) (string, error) {
	result, err := ins.RunCommand(cmd)
	Expect(err).To(BeNil(), "Running command failed: %v\n", err)
	return result, err
}

// PushFile pushes a local file to a GCE instance.
func (ins *Instance) PushFile(srcPath, destPath string) error {
	if ins.ExternalIP == "" {
		ins.populateExternalIP()
	}
	output, err := exec.Command("scp", "-o", "StrictHostKeyChecking no",
		"-i", ins.SshKey,
		srcPath, fmt.Sprintf("%s@%s:%s", ins.SshUser, ins.ExternalIP, destPath)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("Error running scp: %v.\nHere is the output for the command: %v", err, string(output))
	}
	return nil
}

// DeleteInstance deletes a GCE instance.
func (ins *Instance) DeleteInstance() error {
	if _, err := ins.ComputeService.Instances.Get(ins.Project, ins.Zone, ins.Name).Do(); err != nil {
		return err
	}
	if _, err := ins.ComputeService.Instances.Delete(ins.Project, ins.Zone, ins.Name).Do(); err != nil {
		return err
	}
	return nil
}

// runCommand runs the command line specified in cmdLine.
func runCommand(cmdLine []string) (string, error) {
	return runCommandWithinDir(cmdLine, "./")
}

// RunCommandWithinDir runs the command line specified in cmdLine, optionally in
// the directory specified by cmdDir.  In case of failure, we return the
// output to the caller.
func runCommandWithinDir(cmdLine []string, cmdDir string) (string, error) {
	// this is to avoid the cases where there multiple subcommands
	// within a command. For example like docker image pull gcr.io/busybox
	cmdline := []string{
		"/bin/bash",
		"-c",
		strings.Join(cmdLine, " "),
	}
	output, _, err := runCommandCaptureStdout(cmdline, cmdDir)
	return output, err
}

// runCommandCaptureStdout runs the given command line in the same way as
// runCommand but also captures standard output of the command and returns
// it along with exit code and error description
func runCommandCaptureStdout(cmdLine []string, cmdDir string) (string, int, error) {
	var buf bytes.Buffer
	exitCode, err := runCommandWithEnvCaptureStdout(cmdLine, cmdDir, nil, &buf)
	return string(buf.Bytes()), exitCode, err
}

// RunCommandWithEnvCaptureStdout runs the given command line in the same way as
// RunCommandWithEnv but allows to capture standard output into a provided buffer
func runCommandWithEnvCaptureStdout(cmdLine []string, cmdDir string, env []string, buf *bytes.Buffer) (int, error) {
	glog.Infof("Running command: %s in directory %s", strings.Join(cmdLine, " "), cmdDir)
	cmd := exec.Command(cmdLine[0], cmdLine[1:]...)
	cmd.Stdin = os.Stdin
	if buf != nil {
		cmd.Stdout = buf
	} else {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), env...)
	if cmdDir != "" && cmdDir != "." {
		cmd.Dir = cmdDir
	}
	if err := cmd.Run(); err != nil {
		newErr := fmt.Errorf("failed to run command %+v (error: %s)", cmdLine, err)
		glog.Info(newErr)
		return cmd.ProcessState.ExitCode(), newErr
	}
	return 0, nil
}
