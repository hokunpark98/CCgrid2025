package main

import (
	"fmt"
	"sync"
)

// PodStats 구조체 정의 (수신 데이터)
type PodStats struct {
	PodName    string
	Bandwidth  int64
	CPUUtil    float64
	WorkerName string
}

// AverageStats 구조체 정의 (평균 데이터 반환용)
type AverageStats struct {
	TotalBandwidth int64 `json:"TotalBandwidth"`
	AvgCPUUtil     int   `json:"AvgCPUUtil"` // CPU Util을 정수로 저장
}

// 데이터 저장소 구조체 정의
type podDataStore struct {
	sync.RWMutex
	data map[string][]PodStats
}

// 전역 저장소 인스턴스
var store = podDataStore{
	data: make(map[string][]PodStats),
}

// storePodStats 함수: 수신한 프로브 데이터를 최신 100개만 유지하도록 저장
func storePodStats(stats PodStats) {
	store.Lock()
	defer store.Unlock()

	// 파드 이름별로 데이터 저장 및 최대 길이 유지
	podName := stats.PodName
	store.data[podName] = append(store.data[podName], stats)

	// 최대 길이가 100을 초과하면 가장 오래된 데이터 삭제
	if len(store.data[podName]) > 100 {
		store.data[podName] = store.data[podName][1:]
	}
}

// calculateAverages 함수: 모든 Pod에 대해 최근 N개의 데이터 기반 평균 및 대역폭 차이 계산
func calculateAverage(count int) map[string]AverageStats {
	store.RLock()
	defer store.RUnlock()

	// store.data 출력
	fmt.Println("Current store.data contents:")
	for podName, statsList := range store.data {
		fmt.Printf("Pod Name: %s, Stats: %+v\n", podName, statsList)
	}

	averageStatsMap := make(map[string]AverageStats)

	// 각 Pod별로 평균 계산
	for podName, statsList := range store.data {
		if len(statsList) == 0 {
			continue
		}

		// 최근 count 개의 데이터만 사용, count가 statsList 길이보다 길면 statsList 전체 사용
		if count > len(statsList) {
			count = len(statsList)
		}
		recentStats := statsList[len(statsList)-count:]

		// CPU Utilization 평균 계산
		var totalCPUUtil float64
		for _, stats := range recentStats {
			totalCPUUtil += stats.CPUUtil
		}
		avgCPUUtil := int(totalCPUUtil / float64(count)) // 정수로 변환

		// 대역폭 계산 (가장 최신 값 - N번째 최신 값)
		latestBandwidth := recentStats[len(recentStats)-1].Bandwidth
		earliestBandwidth := recentStats[0].Bandwidth
		totalBandwidth := latestBandwidth - earliestBandwidth

		averageStatsMap[podName] = AverageStats{
			TotalBandwidth: totalBandwidth,
			AvgCPUUtil:     avgCPUUtil,
		}
	}

	return averageStatsMap
}
