package main

import (
	"context"
	"fmt"
	"log"
	"math"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// CollectData collects CPU utilization, request counts, and worker node information.
func CollectData(promClient *PrometheusClient, k8sClient *kubernetes.Clientset, namespace string, duration string) (NamespaceData, error) {
	componentPodMap, err := MapServiceToPodInfo(k8sClient, namespace)
	if err != nil {
		return NamespaceData{}, err
	}

	// 파드 이름을 키로 하고 CPU 요청량을 값으로 하는 맵 생성
	cpuRequestMap := make(map[string]int64)
	for _, pods := range componentPodMap {
		for _, pod := range pods {
			cpuRequestMap[pod.PodName] = pod.CpuRequest
		}
	}

	// 네임스페이스 내 모든 파드의 CPU 사용률과 요청 수 가져오기
	cpuUtilizationMap, err := CollectCpuUtilizationForAllPods(promClient, namespace, duration, cpuRequestMap)
	if err != nil {
		return NamespaceData{}, err
	}

	requestCountMap, err := CollectRequestCountForAllPods(promClient, namespace, duration)
	if err != nil {
		return NamespaceData{}, err
	}

	var components []ComponentData
	for serviceName, pods := range componentPodMap {
		var podDataList []PodData
		for _, pod := range pods {
			cpuUtil := cpuUtilizationMap[pod.PodName]
			requestCount := requestCountMap[pod.PodName]
			frequency := workerFrequencies[pod.WorkerNode]

			podDataList = append(podDataList, PodData{
				PodName:        pod.PodName,
				PodIP:          pod.PodIP,
				Port:           pod.Port,
				CpuUtilization: math.Max(float64(cpuUtil), 0),
				RequestCount:   int(math.Max(float64(requestCount), 0)),
				WorkerNode:     pod.WorkerNode,
				Frequency:      frequency,
				CpuRequest:     pod.CpuRequest,
			})
		}

		components = append(components, ComponentData{
			ComponentName: serviceName,
			Pods:          podDataList,
		})
	}

	return NamespaceData{
		Namespace:  namespace,
		Components: components,
	}, nil
}

// MapServiceToPodInfo maps services to their pods in the given namespace.
func MapServiceToPodInfo(k8sClient *kubernetes.Clientset, namespace string) (map[string][]PodData, error) {
	servicePodMap := make(map[string][]PodData)

	services, err := k8sClient.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, service := range services.Items {
		labelSelector := service.Spec.Selector
		if len(labelSelector) > 0 {
			pods, err := k8sClient.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
				LabelSelector: labels.Set(labelSelector).String(),
			})
			if err != nil {
				log.Printf("Failed to list pods for service %s: %v", service.Name, err)
				continue
			}

			// 서비스의 첫 번째 포트를 수신 포트로 설정
			var port int32
			if len(service.Spec.Ports) > 0 {
				port = service.Spec.Ports[0].Port
			}

			for _, pod := range pods.Items {
				cpuRequest := int64(0)
				// 각 컨테이너의 요청된 CPU 리소스를 합산하여 PodData에 저장
				for _, container := range pod.Spec.Containers {
					if container.Resources.Requests != nil {
						if cpuQuantity, ok := container.Resources.Requests["cpu"]; ok {
							cpuRequest += cpuQuantity.MilliValue() // millicores로 변환
						}
					}
				}

				servicePodMap[service.Name] = append(servicePodMap[service.Name], PodData{
					PodName:    pod.Name,
					PodIP:      pod.Status.PodIP,
					Port:       port,
					WorkerNode: pod.Spec.NodeName,
					CpuRequest: cpuRequest, // CPU 요청 값 추가
				})
			}
		}
	}

	return servicePodMap, nil
}

// CollectCpuUtilizationForAllPods collects CPU utilization for all pods in the namespace.
func CollectCpuUtilizationForAllPods(promClient *PrometheusClient, namespace string, duration string, cpuRequestMap map[string]int64) (map[string]float64, error) {
	query := fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace="%s"}[%s])) by (pod) * 100`, namespace, duration)
	result, err := promClient.Query(query)
	if err != nil {
		return nil, err
	}

	cpuUtilizationMap := make(map[string]float64)
	for _, sample := range result {
		podNameInterface, ok := sample.Metric["pod"]
		if !ok {
			continue
		}
		podName := string(podNameInterface)

		// cpuRequestMap에서 podName에 해당하는 CPU 요청량 가져오기
		cpuRequest, exists := cpuRequestMap[podName]
		if !exists || cpuRequest == 0 {
			cpuRequest = 1000 // 기본값으로 1000m 설정
		}

		// 계산식 수정: CPU 사용률 * CPU 요청량 / 1000
		cpuUtilization := float64(sample.Value) * float64(cpuRequest) / 1000
		cpuUtilizationMap[podName] = math.Round(float64(cpuUtilization)*10000) / 10000
	}

	return cpuUtilizationMap, nil
}

// CollectRequestCountForAllPods collects request counts for all pods in the namespace.
func CollectRequestCountForAllPods(promClient *PrometheusClient, namespace string, duration string) (map[string]int, error) {
	query := fmt.Sprintf(`sum(increase(istio_requests_total{kubernetes_namespace="%s", reporter="destination"}[%s])) by (kubernetes_pod_name)`, namespace, duration)
	result, err := promClient.Query(query)
	if err != nil {
		return nil, err
	}

	requestCountMap := make(map[string]int)
	for _, sample := range result {
		podNameInterface, ok := sample.Metric["kubernetes_pod_name"]
		if !ok {
			continue
		}
		podName := string(podNameInterface)
		requestCountMap[podName] = int(math.Round(float64(sample.Value)))
	}

	return requestCountMap, nil
}

// CollectServiceDependencies 함수
// CollectServiceDependencies collects service dependencies without request count
func CollectServiceDependencies(promClient *PrometheusClient, namespace string, duration string) ([]DependencyData, error) {
	query := fmt.Sprintf(`sum(increase(istio_requests_total{kubernetes_namespace="%s"}[%s])) by (source_canonical_service, destination_canonical_service)`, namespace, duration)

	result, err := promClient.Query(query)
	if err != nil {
		return nil, err
	}

	dependencyMap := make(map[string]*DependencyData)

	for _, sample := range result {
		source := string(sample.Metric["source_canonical_service"])
		destination := string(sample.Metric["destination_canonical_service"])

		// 불필요한 서비스 필터링 (예: unknown)
		if source == "unknown" || destination == "unknown" {
			continue
		}

		key := source + "->" + destination

		// 중복 방지를 위해 키를 맵에 추가
		if _, exists := dependencyMap[key]; !exists {
			dependencyMap[key] = &DependencyData{
				Source:      source,
				Destination: destination,
			}
		}
	}

	// 맵을 슬라이스로 변환하여 반환
	var dependencies []DependencyData
	for _, dep := range dependencyMap {
		dependencies = append(dependencies, *dep)
	}

	return dependencies, nil
}
