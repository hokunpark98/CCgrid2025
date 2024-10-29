package trafficAllocator

import (
	"traffic-scheduler/graphGenerator"
	"traffic-scheduler/metricCollector"
)

// ProportionMap stores traffic proportions between source and destination components.
type ProportionMap struct {
	Components map[string]map[string][]SourcePodData `json:"components"`
}

// SourcePodData stores the traffic proportion for multiple destination pods for each source pod.
type SourcePodData struct {
	SourcePodName   string                         `json:"source_pod_name"`
	ProportionDatas []DestinationPodProportionData `json:"proportion_datas"`
}

// DestinationPodProportionData represents the traffic proportion assigned to each destination pod.
type DestinationPodProportionData struct {
	DestinationPodName string `json:"destination_pod_name"`
	Proportion         int    `json:"proportion"` // Percentage
}

// TrafficAllocation calculates traffic allocation using the default Round Robin approach.
func TrafficAllocation(namespace string, componentGraph *graphGenerator.ComponentGraph, componentPodMap *metricCollector.ComponentPodMap) *ProportionMap {
	proportionMap := &ProportionMap{
		Components: make(map[string]map[string][]SourcePodData),
	}

	for sourceComponent, destinationComponents := range componentGraph.Components {
		for _, destinationComponent := range destinationComponents {
			sourcePodDataList := calProportionComponentPair(sourceComponent, destinationComponent, componentPodMap)

			if proportionMap.Components[sourceComponent] == nil {
				proportionMap.Components[sourceComponent] = make(map[string][]SourcePodData)
			}
			proportionMap.Components[sourceComponent][destinationComponent] = sourcePodDataList
		}
	}

	// Lua script generation
	GenerateLuaScript(namespace, proportionMap, componentPodMap)
	return proportionMap
}

// 그냥 공정하게만
func calProportionComponentPair(sourceComponentName, destinationComponentName string, componentPodMap *metricCollector.ComponentPodMap) []SourcePodData {
	sourcePods := componentPodMap.Components[sourceComponentName]
	destinationPods := componentPodMap.Components[destinationComponentName]

	numDestinationPods := len(destinationPods)
	proportionPerPod := 100 / numDestinationPods
	remainder := 100 % numDestinationPods

	var sourcePodDataList []SourcePodData

	for _, sourcePod := range sourcePods {
		var proportionDataList []DestinationPodProportionData
		accumulatedProportion := 0

		for i, destinationPod := range destinationPods {
			proportion := proportionPerPod
			if i < remainder {
				proportion += 1 // 나머지를 첫 번째 몇 개의 Pod에 배분합니다.
			}
			accumulatedProportion += proportion

			proportionDataList = append(proportionDataList, DestinationPodProportionData{
				DestinationPodName: destinationPod.PodName,
				Proportion:         accumulatedProportion,
			})
		}

		sourcePodDataList = append(sourcePodDataList, SourcePodData{
			SourcePodName:   sourcePod.PodName,
			ProportionDatas: proportionDataList,
		})
	}

	return sourcePodDataList
}
