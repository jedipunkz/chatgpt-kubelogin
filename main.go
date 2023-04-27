package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ktr0731/go-fuzzyfinder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/homedir"
)

type PodGetter interface {
	GetPods(clientset kubernetes.Interface) ([]corev1.Pod, error)
}

type PodExecutor interface {
	ExecInPod(clientset *kubernetes.Clientset, config *rest.Config, podName string, namespace string) error
}

type podGetterImpl struct{}

func (p *podGetterImpl) GetPods(clientset kubernetes.Interface) ([]corev1.Pod, error) {
	podList, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

type podExecutorImpl struct{}

func (p *podExecutorImpl) ExecInPod(clientset kubernetes.Interface, config *rest.Config, podName string, namespace string) error {
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("stdin", "true").
		Param("stdout", "true").
		Param("stderr", "true").
		Param("tty", "true").
		Param("command", "/bin/sh")

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    true,
	})
	if err != nil {
		return err
	}

	return nil
}

func main() {
	kubeconfigPath := filepath.Join(homedir.HomeDir(), ".kube", "config")

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		fmt.Printf("Error building kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("Error creating clientset: %v\n", err)
		os.Exit(1)
	}

	podGetter := &podGetterImpl{}
	podList, err := podGetter.GetPods(clientset)
	if err != nil {
		fmt.Printf("Error getting pods: %v\n", err)
		os.Exit(1)
	}

	index, err := fuzzyfinder.Find(podList, func(i int) string {
		return podList[i].Name
	})

	if err != nil {
		fmt.Printf("Error finding pod: %v\n", err)
		os.Exit(1)
	}

	selectedPod := podList[index]
	fmt.Printf("Logging into pod %s in namespace %s...\n", selectedPod.Name, selectedPod.Namespace)
	podExecutor := &podExecutorImpl{}
	if err := podExecutor.ExecInPod(clientset, config, selectedPod.Name, selectedPod.Namespace); err != nil {
		fmt.Printf("Error executing command in pod: %v\n", err)
	}
}
