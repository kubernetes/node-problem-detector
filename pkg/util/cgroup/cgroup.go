/*
Copyright 2021 The Kubernetes Authors All rights reserved.

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

package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/containerd/cgroups"
	cgroupsV1 "github.com/containerd/cgroups/stats/v1"
	v1 "k8s.io/api/core/v1"
)

const (
	kubeCFSBasePath = "/sys/fs/cgroup/cpu/kubepods"
)

var (
	bestEffortPath = filepath.Join(kubeCFSBasePath, "besteffort")
	burstablePath  = filepath.Join(kubeCFSBasePath, "burstable")

	podCaptureRe = regexp.MustCompile(`(?m)\/pod(.*)\/(.*)$`)
)

type podCFS struct {
	PodID       string
	ContainerID string
	QOSClass    v1.PodQOSClass
}

func (p *podCFS) String() {
	fmt.Printf("Pod = %s, Container = %s, QOS = %s\n", p.PodID, p.ContainerID, p.QOSClass)
}

func allPodPaths() ([]string, error) {
	podPaths := []string{}
	err := filepath.Walk(kubeCFSBasePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && strings.Contains(path, "/pod") {
			podPaths = append(podPaths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return podPaths, nil
}

// AllKubeCgroups lists and returns info about pods and containers tracked
func AllKubeCgroups() ([]podCFS, error) {
	podPaths, err := allPodPaths()
	if err != nil {
		return nil, err
	}

	pods := []podCFS{}
	for _, p := range podPaths {
		matches := podCaptureRe.FindStringSubmatch(p)
		if len(matches) != 3 {
			// We expect [full match, podID, containerID]
			continue
		}
		qos := v1.PodQOSGuaranteed
		if strings.Contains(p, "besteffort") {
			qos = v1.PodQOSBestEffort
		} else if strings.Contains(p, "burstable") {
			qos = v1.PodQOSBurstable
		}
		pods = append(pods, podCFS{
			PodID:       matches[1],
			ContainerID: matches[2],
			QOSClass:    qos,
		})
	}
	return pods, err
}

func (p *podCFS) CgroupPath() string {
	kubepath := "kubepods/burstable/pod" + p.PodID + "/" + p.ContainerID
	if p.QOSClass == "guaranteed" {
		kubepath = "kubepods/pod" + p.PodID + "/" + p.ContainerID
	} else if p.QOSClass == "besteffort" {
		kubepath = "kubepods/besteffort/pod" + p.PodID + "/" + p.ContainerID
	}
	return kubepath
}

func (p *podCFS) CgroupStats() (*cgroupsV1.Metrics, error) {
	control, err := cgroups.Load(cgroups.V1, cgroups.StaticPath(p.CgroupPath()))
	if err != nil {
		panic(err)
	}
	return control.Stat()
}
