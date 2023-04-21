package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ktr0731/go-fuzzyfinder"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func getPods(clientset *kubernetes.Clientset) ([]corev1.Pod, error) {
	podList, err := clientset.CoreV1().Pods("").List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return podList.Items, nil
}

func filterPods(pods []corev1.Pod, filter string) []corev1.Pod {
	var filteredPods []corev1.Pod
	for _, pod := range pods {
		if strings.Contains(pod.Name, filter) {
			filteredPods = append(filteredPods, pod)
		}
	}
	return filteredPods
}

func execInPod(podName string, namespace string, kubeconfig string) error {
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig, "exec", "-it", "-n", namespace, podName, "--", "/bin/sh", "-c", fmt.Sprintf("export PS1='[Pod: %s] \\u@\\h:\\w\\$ '; exec /bin/sh", podName))

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
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

	podList, err := getPods(clientset)
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
	fmt.Printf("Logging into pod %s...\n", selectedPod.Name)
	if err := execInPod(selectedPod.Name, selectedPod.Namespace, kubeconfigPath); err != nil {
		fmt.Printf("Error executing command in pod: %v\n", err)
	}

}
