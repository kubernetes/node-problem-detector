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
	"fmt"
	"os/exec"
	"time"

	"k8s.io/node-problem-detector/test/e2e/lib/ssh"

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
		return instance, fmt.Errorf("failed to get project info %q", instance.Project)
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
	for i := 0; i < 30 && !instanceRunning; i++ {
		if i > 0 {
			time.Sleep(time.Second * 20)
		}
		if instance.ExternalIP == "" {
			instance.populateExternalIP()
		}

		result := ssh.Run("pwd", instance.ExternalIP, instance.SshUser, instance.SshKey)
		if result.SSHError != nil {
			err = fmt.Errorf("SSH to instance %s failed: %s", instance.Name, result.SSHError)
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
func (ins *Instance) RunCommand(cmd string) ssh.Result {
	if ins.ExternalIP == "" {
		ins.populateExternalIP()
	}
	return ssh.Run(cmd, ins.ExternalIP, ins.SshUser, ins.SshKey)
}

// PushFile pushes a local file to a GCE instance.
func (ins *Instance) PushFile(srcPath, destPath string) error {
	if ins.ExternalIP == "" {
		ins.populateExternalIP()
	}
	return exec.Command("scp", "-o", "StrictHostKeyChecking no",
		"-i", ins.SshKey,
		srcPath, fmt.Sprintf("%s@%s:%s", ins.SshUser, ins.ExternalIP, destPath)).Run()
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
