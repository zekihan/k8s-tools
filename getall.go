package main

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func nodes() {
	k8sClient, err := getK8sClient(*kubeconfig)
	if err != nil {
		panic(fmt.Sprintf("Error getting Kubernetes client: %v", err))
	}
	nodes, err := k8sClient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		panic(fmt.Sprintf("Error listing RuntimeClasses: %v", err))
	}
	for _, node := range nodes.Items {
		fmt.Printf("Node Name: %s\n", node.Name)
	}
}

func getK8sClient(kubeconfigPath string) (*kubernetes.Clientset, error) {
	config, err := getConfig(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("error building kubeconfig: %v", err)
	}

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("error creating Kubernetes client: %v", err)
	}
	return clientset, nil
}

func getConfig(kubeconfigPath string) (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if kubeconfigPath != "" {
		loadingRules.ExplicitPath = kubeconfigPath
	}
	configOverrides := &clientcmd.ConfigOverrides{
		CurrentContext: *explicitContext,
	}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("error building kubeconfig: %v", err)
	}
	return config, nil
}
