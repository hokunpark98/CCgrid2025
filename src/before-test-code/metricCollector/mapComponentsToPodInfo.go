package metricCollector

import (
	"context"
	"fmt"
	"log"
	"traffic-scheduler/prometheusClient"

	"os"
	"path/filepath"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// PodInfo는 파드의 이름, IP 주소, 호스트 이름, 호스트 IP 주소 및 포트를 나타냅니다.
type PodInfo struct {
	PodName   string   `json:"pod_name"`
	PodIP     string   `json:"pod_ip"`
	HostName  string   `json:"host_name"`
	HostIP    string   `json:"host_ip"`
	Component string   `json:"component"`
	Ports     []string `json:"ports"` // 여러 포트를 저장할 수 있는 필드
}

// ComponentPodMap은 컴포넌트와 파드의 매핑을 저장하는 맵을 나타냅니다.
type ComponentPodMap struct {
	Components map[string][]PodInfo `json:"components"`
	Pods       map[string]PodInfo   `json:"pods"`
}

// getK8sClient는 클러스터 내부 또는 외부에서 Kubernetes 클라이언트를 생성합니다.
func getK8sClient() (*kubernetes.Clientset, error) {
	// 먼저 InClusterConfig를 시도
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

// MapComponentToPodInfo 함수는 주어진 네임스페이스에서 각 컴포넌트에 대응하는 서비스의 포트 정보를 반환합니다.
func MapComponentToPodInfo(promClient *prometheusClient.PrometheusClient, namespace string) (*ComponentPodMap, error) {
	// Kubernetes 클라이언트 생성
	k8sClient, err := getK8sClient()
	if err != nil {
		log.Fatalf("Error creating Kubernetes client: %v", err)
		return nil, err
	}

	componentsPodMap := &ComponentPodMap{
		Components: make(map[string][]PodInfo),
		Pods:       make(map[string]PodInfo),
	}

	// Prometheus 쿼리를 통해 네임스페이스 내의 모든 컴포넌트를 가져옴
	componentQuery := `kube_deployment_labels{namespace="` + namespace + `"}`
	componentResult, err := promClient.Query(componentQuery)
	if err != nil {
		log.Fatalf("Error querying Prometheus for components: %v", err)
		return nil, err
	}

	// 각 컴포넌트(label_app)를 순회하면서 해당 컴포넌트에 속한 서비스 정보를 가져옴
	for _, sample := range componentResult {
		component := string(sample.Metric["label_app"])

		query := `kube_pod_labels{label_app="` + component + `", namespace="` + namespace + `"}`

		labelResult, err := promClient.Query(query)
		if err != nil {
			log.Fatalf("Error querying Prometheus for labels: %v", err)
			return nil, err
		}

		for _, sample := range labelResult {
			podName := string(sample.Metric["pod"])

			// Kubernetes API를 사용하여 서비스 정보 가져오기
			svc, err := k8sClient.CoreV1().Services(namespace).Get(context.TODO(), component, metav1.GetOptions{})
			if err != nil {
				log.Fatalf("Error getting service info from Kubernetes API: %v", err)
				return nil, err
			}

			var podPorts []string
			for _, port := range svc.Spec.Ports {
				// 서비스의 포트 번호를 문자열로 변환하여 저장
				portStr := fmt.Sprintf("%d", port.Port)
				podPorts = append(podPorts, portStr)
			}

			query := `kube_pod_info{namespace="` + namespace + `", pod="` + podName + `"}`

			infoResult, err := promClient.Query(query)
			if err != nil {
				log.Fatalf("Error querying Prometheus for pod info: %v", err)
				return nil, err
			}

			for _, sample := range infoResult {
				podInfo := PodInfo{

					PodName:   string(sample.Metric["pod"]),
					PodIP:     string(sample.Metric["pod_ip"]), // 서비스의 ClusterIP 사용
					HostName:  string(sample.Metric["node"]),
					HostIP:    string(sample.Metric["host_ip"]),
					Ports:     podPorts, // 서비스에서 가져온 포트 정보
					Component: component,
				}

				componentsPodMap.Components[component] = append(componentsPodMap.Components[component], podInfo)
				componentsPodMap.Pods[podName] = podInfo
			}
		}
	}

	return componentsPodMap, nil
}
