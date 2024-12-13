package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func main() {
	// Prometheus client initialization
	promClient, err := NewPrometheusClient("http://prometheus-service.monitoring.svc.cluster.local:8080")
	//promClient, err := NewPrometheusClient("http://10.107.204.182:8080")
	if err != nil {
		log.Fatalf("Failed to create Prometheus client: %v", err)
	}

	// Kubernetes client initialization
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Printf("InClusterConfig failed, trying Kubeconfig: %v", err)

		// InClusterConfig가 실패하면 Kubeconfig 파일을 사용
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Printf("InClusterConfig failed, trying Kubeconfig: %v", err)

		}

		kubeconfigPath := filepath.Join(homeDir, ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			log.Printf("InClusterConfig failed, trying Kubeconfig: %v", err)

		}
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	// Handler setup
	h := &Handler{
		PromClient: promClient,
		K8sClient:  k8sClient,
	}

	// HTTP server setup
	http.HandleFunc("/metrics", h.MetricsHandler)
	http.HandleFunc("/dag", h.DagHandler)

	log.Println("Server is running on port 21001")
	if err := http.ListenAndServe(":21001", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

}
