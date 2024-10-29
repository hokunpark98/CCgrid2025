package metricCollector

import (
	"log"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// getK8sClient 함수는 클러스터 내부에서는 InClusterConfig를 사용하고,
// 외부에서는 kubeconfig 파일을 사용하여 Kubernetes 클라이언트를 반환합니다.
func GetK8sClient() (*kubernetes.Clientset, error) {
	// 먼저 InClusterConfig를 시도 (클러스터 내부)
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("InClusterConfig failed, trying Kubeconfig: %v", err)

		// InClusterConfig가 실패하면 Kubeconfig 파일을 사용
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		kubeconfigPath := filepath.Join(homeDir, ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, err
		}
	}

	// Kubernetes 클라이언트 생성
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
