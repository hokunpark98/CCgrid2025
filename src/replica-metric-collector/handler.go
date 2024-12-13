package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"k8s.io/client-go/kubernetes"
)

type Handler struct {
	PromClient *PrometheusClient
	K8sClient  *kubernetes.Clientset
}

// MetricsHandler handles HTTP requests and returns data in JSON format.
func (h *Handler) MetricsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse 'value' parameter for duration
	namespace := r.URL.Query().Get("namespace")
	if namespace == "" {
		namespace = "default" // Default duration is 5 seconds
	}

	value := r.URL.Query().Get("value")
	if value == "" {
		value = "5" // Default duration is 5 seconds
	}
	duration, err := strconv.Atoi(value)
	if err != nil || duration <= 0 {
		http.Error(w, "Invalid 'value' parameter", http.StatusBadRequest)
		return
	}
	durationStr := fmt.Sprintf("%ds", duration)

	// Collect data
	data, err := CollectData(h.PromClient, h.K8sClient, namespace, durationStr)
	if err != nil {
		log.Printf("Error collecting data: %v", err)
		http.Error(w, "Error collecting data", http.StatusInternalServerError)
		return
	}

	// Return data as JSON
	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		log.Printf("Error encoding JSON: %v", err)
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
	}
}

func (h *Handler) DagHandler(w http.ResponseWriter, r *http.Request) {
	// 쿼리 파라미터에서 value와 namespace를 가져옵니다.
	value := r.URL.Query().Get("value")
	namespace := r.URL.Query().Get("namespace")
	if value == "" || namespace == "" {
		http.Error(w, "Missing 'value' or 'namespace' parameter", http.StatusBadRequest)
		return
	}

	// 서비스 의존성 수집
	dependencies, err := CollectServiceDependencies(h.PromClient, namespace, value+"s")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error collecting dependencies: %v", err), http.StatusInternalServerError)
		return
	}

	// JSON 응답 반환
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dependencies)
}
