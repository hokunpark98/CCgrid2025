package main

import (
	"log"
	"net/http"
	"traffic-scheduler/prometheusClient"
)

func main() {
	// Prometheus 클라이언트 생성
	promClient, err := prometheusClient.NewPrometheusClient("http://10.102.138.205:8080")
	if err != nil {
		log.Fatalf("Failed to create Prometheus client: %v", err)
	}

	// 요청 처리 핸들러 설정
	http.HandleFunc("/get-graph", func(w http.ResponseWriter, r *http.Request) {
		handleGetGraph(w, r, promClient)
	})

	http.HandleFunc("/get-monitoring-info", func(w http.ResponseWriter, r *http.Request) {
		handleGetMonitoringInfo(w, r, promClient)
	})

	http.HandleFunc("/get-node-cpu-hz", func(w http.ResponseWriter, r *http.Request) {
		handleGetNodeCpuHz(w, promClient)
	})

	http.HandleFunc("/traffic-schedule", func(w http.ResponseWriter, r *http.Request) {
		handleTrafficSchedule(w, r, promClient)
	})
	// 서버 시작
	log.Println("Server starting on port 13000")
	if err := http.ListenAndServe(":13000", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
