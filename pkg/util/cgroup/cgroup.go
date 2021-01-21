package cgroup

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/containerd/cgroups"
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
	podID       string
	containerID string
	QOSClass    v1.PodQOSClass
}

func (p *podCFS) String() {
	fmt.Printf("Pod = %s, Container = %s, QOS = %s\n", p.podID, p.containerID, p.QOSClass)
}

// TODO: add a test for this
var allPodPaths = func() ([]string, error) {
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

// TODO: add a test for this
// Recursively lists and returns info about pods and containers tracked
func allKubeCgroups() ([]podCFS, error) {
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
			podID:       matches[1],
			containerID: matches[2],
			QOSClass:    qos,
		})
	}
	return pods, err
}

func main() {
	allPods, err := allKubeCgroups()
	if err != nil {
		panic(err)
	}

	counter := make(map[string]int)
	for _, p := range allPods {
		counter[p.podID]++
		p.String()
	}
	fmt.Printf("num of pods = %d", len(counter))
	fmt.Printf("num of containers = %d", len(allPods))

	_, err = cgroups.Load(cgroups.V1, cgroups.RootPath)
	if err != nil {
		panic(err)
	}
	// stats, err := control.Stat()
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("stats = %+v", stats)
}
