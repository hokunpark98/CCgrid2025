package metricCollector

import (
	"fmt"
	"math"
	"traffic-scheduler/graphGenerator"
	"traffic-scheduler/prometheusClient"
)

// PodRequestDurationData는 특정 파드의 평균 요청 지속 시간을 나타냅니다.
type PodRequestDurationData struct {
	PodName         string `json:"pod_name"`
	RequestDuration int    `json:"request_duration"` // ms 단위
}

// ComponentRequestDurationData는 특정 컴포넌트와 그 컴포넌트의 파드들이 수신한 평균 요청 지속 시간을 나타냅니다.
type ComponentRequestDurationData struct {
	ComponentName string                   `json:"component_name"`
	PodDurations  []PodRequestDurationData `json:"pod_durations"`
}

// RequestDurationData는 전체 평균 요청 지속 시간 데이터를 포함합니다.
type RequestDurationMap struct {
	Components map[string]ComponentRequestDurationData `json:"components"`
}

// CollectRequestDurationPerPod는 컴포넌트 그래프와 컴포넌트-파드 매핑을 기반으로 각 파드의 평균 요청 지속 시간을 수집합니다.
func CollectRequestDurationPerPod(promClient *prometheusClient.PrometheusClient, namespace string, componentGraph *graphGenerator.ComponentGraph, componentPodMap *ComponentPodMap, duration string, requestCountMap *RequestCountMap) (*RequestDurationMap, error) {
	requestData := &RequestDurationMap{
		Components: make(map[string]ComponentRequestDurationData),
	}

	for component := range componentGraph.Components {
		for _, link := range componentGraph.Components[component] {
			uc := component
			dc := link

			dcPods, exists := componentPodMap.Components[dc]
			if !exists {
				continue
			}

			var podDurationList []PodRequestDurationData

			for _, pod := range dcPods {
				// `istio_duration_total` 쿼리를 실행하여 총 지속 시간을 가져옴
				query := fmt.Sprintf(`increase(istio_request_duration_milliseconds_sum{kubernetes_namespace="%s", kubernetes_pod_name="%s", source_app="%s"}[%s])`,
					namespace, pod.PodName, uc, duration)

				result, err := promClient.Query(query)
				if err != nil {
					return nil, err
				}

				var totalDuration float64
				for _, sample := range result {
					totalDuration += float64(sample.Value)
				}

				// 해당 파드의 요청 수를 requestCountMap에서 찾아서 평균 지속 시간 계산
				var totalRequests int
				if componentData, exists := requestCountMap.Components[dc]; exists {
					totalRequests = componentData.PodRequestMap[pod.PodName].RequestCount
				}

				if totalRequests > 0 {
					averageDuration := (totalDuration / float64(totalRequests))
					roundedDuration := int(math.Round(averageDuration))

					podDurationList = append(podDurationList, PodRequestDurationData{
						PodName:         pod.PodName,
						RequestDuration: roundedDuration,
					})
				}
			}

			if existingComponent, exists := requestData.Components[dc]; exists {
				existingComponent.PodDurations = append(existingComponent.PodDurations, podDurationList...)
				requestData.Components[dc] = existingComponent
			} else {
				requestData.Components[dc] = ComponentRequestDurationData{
					ComponentName: dc,
					PodDurations:  podDurationList,
				}
			}
		}
	}

	return requestData, nil
}
