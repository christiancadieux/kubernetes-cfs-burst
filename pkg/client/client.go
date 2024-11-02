package client

import (
	"fmt"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
)

const (
	KUBECONFIG = "/etc/kubernetes/kubeconfig"
)

func LoadKubeconfigClient() (*kubernetes.Clientset, error) {

	config, err := clientcmd.BuildConfigFromFlags("", KUBECONFIG)
	if err != nil {
		fmt.Printf("error getting Kubernetes config: %v\n", err)
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("NewForConfig - %v", err)
	}
	return clientset, nil

}

func LoadInClusterClient() (*kubernetes.Clientset, error) {

	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("NewForConfig - %v", err)
	}
	return clientset, nil

}
