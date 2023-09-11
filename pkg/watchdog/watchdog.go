package controller

import (
	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v12 "k8s.io/node-problem-detector/pkg/watchdog/api/ecx/v1"
	"k8s.io/node-problem-detector/pkg/watchdog/client/clientset/versioned"
	"k8s.io/node-problem-detector/pkg/watchdog/client/informers/externalversions"
	v1 "k8s.io/node-problem-detector/pkg/watchdog/client/informers/externalversions/ecx/v1"
	"os"
	"time"
)

// PodCache contains a podInformer of pod
type SelfHealingTaskInstanceCache struct {
	selfHealingTaskInstanceInformer v1.SelfHealingTaskInstanceInformer
}

var (
	selfHealingTaskInstanceCacheCache *SelfHealingTaskInstanceCache
)

// NewPodCache creates a new podCache
func NewSelfHealingTaskInstanceCache(client versioned.Interface) {
	selfHealingTaskInstanceCacheCache = new(SelfHealingTaskInstanceCache)

	factory := externalversions.NewSharedInformerFactoryWithOptions(client, time.Minute,
		externalversions.WithTweakListOptions(func(options *metav1.ListOptions) {
		}))
	selfHealingTaskInstanceCacheCache.selfHealingTaskInstanceInformer = factory.Ecx().V1().SelfHealingTaskInstances()

	ch := make(chan struct{})
	go selfHealingTaskInstanceCacheCache.selfHealingTaskInstanceInformer.Informer().Run(ch)

	for !selfHealingTaskInstanceCacheCache.selfHealingTaskInstanceInformer.Informer().HasSynced() {
		time.Sleep(time.Second)
	}
	glog.Infof("selfHealingTaskInstanceCacheCache cache is running")
}

func GetActiveSelfHealingTaskInstance() map[string]*v12.SelfHealingTaskInstance {
	if selfHealingTaskInstanceCacheCache == nil {
		return nil
	}

	activeInstances := make(map[string]*v12.SelfHealingTaskInstance)

	for _, item := range selfHealingTaskInstanceCacheCache.selfHealingTaskInstanceInformer.Informer().GetStore().List() {
		instance, ok := item.(*v12.SelfHealingTaskInstance)
		if !ok {
			continue
		}

		if taskIsTerminated(instance) {
			continue
		}

		if !IsNodeRequireInstance(instance) {
			continue
		}

		activeInstances[string(instance.UID)] = instance
	}

	return activeInstances
}

//func GetPodForLister(namespace, name string) (*v1.Pod, error) {
//	pod, err := podCache.podInformer.Lister().Pods(namespace).Get(name)
//	if err != nil {
//		return nil, err
//	}
//
//	if podIsTerminated(pod) {
//		return nil, fmt.Errorf("terminated pod")
//	}
//
//	if !utils.IsGPURequiredPod(pod) {
//		return nil, fmt.Errorf("no gpu pod")
//	}
//
//	return pod, nil
//}

func taskIsTerminated(instance *v12.SelfHealingTaskInstance) bool {
	return instance.DeletionTimestamp != nil
}

func IsNodeRequireInstance(instance *v12.SelfHealingTaskInstance) bool {
	return instance.Status.SelfHealingCheckStatus.NodeName == getNodeName()
}

func getNodeName() string {

	nodeName := os.Getenv("NODE_NAME")
	if nodeName != "" {
		return nodeName
	}

	// For backward compatibility. If the env is not set, get the hostname
	// from os.Hostname(). This may not work for all configurations and
	// environments.
	nodeName, err := os.Hostname()
	if err != nil {
		return ""
	}

	return nodeName
}
