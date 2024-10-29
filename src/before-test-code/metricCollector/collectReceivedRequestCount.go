package metricCollector

import (
	"fmt"
	"math"
	"traffic-scheduler/graphGenerator"
	"traffic-scheduler/prometheusClient"
)

// PodRequestCountData는 특정 파드가 수신한 요청 수를 나타냅니다.
type PodRequestCountData struct {
	PodName      string `json:"pod_name"`
	RequestCount int    `json:"request_count"`
}

// ComponentRequestCountData는 특정 컴포넌트와 그 컴포넌트의 파드들이 수신한 요청 수를 나타냅니다.
type ComponentRequestCountData struct {
	ComponentName string                          `json:"component_name"`
	PodRequestMap map[string]*PodRequestCountData `json:"pod_request_map"` // PodName을 키로 사용하는 맵
}

// RequestCountData는 전체 요청 수 데이터를 포함합니다.
type RequestCountMap struct {
	Components map[string]ComponentRequestCountData `json:"components"`
}

// CollectRequestCountPerPod는 컴포넌트 그래프와 컴포넌트-파드 매핑을 기반으로 각 파드의 수신 요청 수를 수집합니다.
func CollectRequestCountPerPod(promClient *prometheusClient.PrometheusClient, namespace string, componentGraph *graphGenerator.ComponentGraph, componentPodMap *ComponentPodMap, duration string) (*RequestCountMap, error) {
	requestData := &RequestCountMap{
		Components: make(map[string]ComponentRequestCountData),
	}

	// 모든 파드의 요청 수를 0으로 초기화하고 PodRequestMap도 초기화
	for component, pods := range componentPodMap.Components {
		podRequestMap := make(map[string]*PodRequestCountData)
		for _, pod := range pods {
			podRequestData := PodRequestCountData{
				PodName:      pod.PodName,
				RequestCount: 0,
			}
			podRequestMap[pod.PodName] = &podRequestData
		}
		requestData.Components[component] = ComponentRequestCountData{
			ComponentName: component,
			PodRequestMap: podRequestMap,
		}
	}

	// 각 링크에 대해 요청 수를 수집
	for component := range componentGraph.Components {
		for _, link := range componentGraph.Components[component] {
			uc := component
			dc := link

			dcPods, exists := componentPodMap.Components[dc]
			if !exists {
				continue
			}

			for _, pod := range dcPods {
				// namespace, dc replica, upstream component과 기간에 따라 쿼리 생성
				query := fmt.Sprintf(`increase(istio_requests_total{kubernetes_namespace="%s", kubernetes_pod_name="%s", source_app="%s", reporter="destination", destination_service_name="PassthroughCluster"}[%s])`,
					namespace, pod.PodName, uc, duration)
				fmt.Printf("increase(istio_requests_total{kubernetes_namespace=\"%s\", kubernetes_pod_name=\"%s\", source_app=\"%s\", reporter=\"destination\"}[%s]", namespace, pod.PodName, uc, duration)
				result, err := promClient.Query(query)
				if err != nil {
					return nil, err
				}

				var totalRequests float64
				for _, sample := range result {
					totalRequests += float64(sample.Value)
				}

				// 소수점을 첫째 자리에서 반올림하고 정수로 변환
				roundedRequests := int(math.Round(totalRequests))

				// PodRequestMap을 사용해 요청 수 업데이트
				existingComponent := requestData.Components[dc]
				if podData, found := existingComponent.PodRequestMap[pod.PodName]; found {
					podData.RequestCount += roundedRequests
				}
			}
		}
	}

	return requestData, nil
}
