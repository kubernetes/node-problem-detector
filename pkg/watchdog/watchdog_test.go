package controller

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/node-problem-detector/pkg/watchdog/client/clientset/versioned"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func Test_watchdog(t *testing.T) {

	GetConfig := config.GetConfig

	rest, err := GetConfig()

	//var kubeconfig = "E:\\kubeconfig\\qujing1.txt"
	//config, err := clientcmd.BuildConfigFromFlags("", rest)
	//if err != nil {
	//	fmt.Println(err)
	//
	//}

	clisetK8s, err := kubernetes.NewForConfig(rest)

	nodes, err := clisetK8s.CoreV1().Nodes().List(metav1.ListOptions{})

	fmt.Println(nodes)

	clientset, err := versioned.NewForConfig(rest)
	if err != nil {
		fmt.Println(err)
		return
	}

	//instances, err := clientset.EcxV1().SelfHealingTaskInstances("deafult").List(metav1.ListOptions{})
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}

	NewSelfHealingTaskInstanceCache(clientset)

	GetActiveSelfHealingTaskInstance()

}
