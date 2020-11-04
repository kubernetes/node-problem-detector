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

package gke

import (
	"cloud.google.com/go/compute/metadata"
	"github.com/golang/glog"
	"github.com/shirou/gopsutil/host"
	"k8s.io/node-problem-detector/pkg/util"
)

type Metadata struct {
	ProjectID     string `json:"projectID"`
	Location      string `json:"location"`
	ClusterName   string `json:"clusterName"`
	InstanceID    string `json:"instanceID"`
	NodeName      string `json:"nodeName"`
	OSVersion     string `json:"osVersion"`
	KernelVersion string `json:"kernelVersion"`
}

func (md *Metadata) HasMissingField() bool {
	return md.ProjectID == "" || md.Location == "" || md.InstanceID == "" || md.NodeName == "" || md.ClusterName == "" || md.OSVersion == "" || md.KernelVersion == ""
}

func (md *Metadata) Populate() error {
	var err error
	glog.Info("Fetching GCE metadata from metadata server")
	if md.ProjectID == "" {
		md.ProjectID, err = metadata.ProjectID()
		if err != nil {
			return err
		}
	}
	if md.Location == "" {
		md.Location, err = metadata.Zone()
		if err != nil {
			return err
		}
	}
	if md.InstanceID == "" {
		md.InstanceID, err = metadata.InstanceID()
		if err != nil {
			return err
		}
	}
	if md.NodeName == "" {
		md.NodeName, err = metadata.InstanceName()
		if err != nil {
			return err
		}
	}

	if md.ClusterName == "" {
		md.ClusterName, err = metadata.Get("instance/attributes/cluster-name")
		if err != nil {
			return err
		}
	}
	if md.OSVersion == "" {
		md.OSVersion, err = util.GetOSVersion()
		if err != nil {
			return err
		}
	}
	if md.KernelVersion == "" {
		md.KernelVersion, err = host.KernelVersion()
		if err != nil {
			return err
		}
	}

	return nil
}
