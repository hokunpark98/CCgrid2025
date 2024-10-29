package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

// /probe 엔드포인트 핸들러 (매초 데이터 수신)
func probeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var podStatsMap map[string]PodStats
	if err := json.NewDecoder(r.Body).Decode(&podStatsMap); err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 프로브 데이터 저장
	for _, stats := range podStatsMap {
		storePodStats(stats)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Probe data received successfully")
}

// /metrics 엔드포인트 핸들러 (평균 제공)
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	// value 파라미터 가져오기
	value := r.URL.Query().Get("value")
	count, err := strconv.Atoi(value)
	if err != nil || count <= 0 {
		http.Error(w, "Invalid 'value' parameter", http.StatusBadRequest)
		return
	}

	// 모든 파드에 대해 평균 계산
	averageStatsMap := calculateAverage(count)

	// JSON 응답 반환
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(averageStatsMap); err != nil {
		http.Error(w, "Failed to encode JSON", http.StatusInternalServerError)
	}
}
