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

package gce

import (
	"cloud.google.com/go/compute/metadata"
	"github.com/golang/glog"
)

type Metadata struct {
	ProjectID    string `json:"projectID"`
	Zone         string `json:"zone"`
	InstanceID   string `json:"instanceID"`
	InstanceName string `json:"instanceName"`
}

func (md *Metadata) HasMissingField() bool {
	if md.ProjectID == "" || md.Zone == "" || md.InstanceID == "" || md.InstanceName == "" {
		return true
	}
	return false
}

func (md *Metadata) PopulateFromGCE() error {
	var err error
	glog.Info("Fetching GCE metadata from metadata server")
	if md.ProjectID == "" {
		md.ProjectID, err = metadata.ProjectID()
		if err != nil {
			return err
		}
	}
	if md.Zone == "" {
		md.Zone, err = metadata.Zone()
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
	if md.InstanceName == "" {
		md.InstanceName, err = metadata.InstanceName()
		if err != nil {
			return err
		}
	}

	return nil
}
