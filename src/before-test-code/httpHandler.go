package main

import (
	"fmt"
	"log"
	"net/http"
	"traffic-scheduler/graphGenerator"
	"traffic-scheduler/logging"
	"traffic-scheduler/metricCollector"
	"traffic-scheduler/prometheusClient"
	"traffic-scheduler/trafficAllocator"
)

// handleGetGraph는 /get-graph 요청을 처리하고 컴포넌트 그래프를 생성하여 응답
func handleGetGraph(w http.ResponseWriter, r *http.Request, promClient *prometheusClient.PrometheusClient) {
	namespace := r.URL.Query().Get("namespace")

	if namespace == "" {
		logging.Alert(w, "fail\n")
		return
	}

	logFile := logging.GenerateLogFile()
	defer logFile.Close()

	componentGraph, err := graphGenerator.GenerateGraph(promClient, namespace)
	if err != nil {
		logMessage := fmt.Sprintf("Failed to generate graph: %v\n", err)
		logging.LogMessage(logFile, logMessage)
		logging.Alert(w, logMessage)
		return
	}
	logging.LogComponentGraph(componentGraph, logFile)

	fmt.Printf("Component Graph: %+v\n", componentGraph)
}

func handleGetComponentGraph(w http.ResponseWriter, r *http.Request, promClient *prometheusClient.PrometheusClient) {
	namespace := r.URL.Query().Get("namespace")

	if namespace == "" {
		logging.Alert(w, "fail\n")
		return
	}

	logFile := logging.GenerateLogFile()
	defer logFile.Close()
	componentPodMap, err := metricCollector.MapComponentToPodInfo(promClient, namespace)
	if err != nil {
		logMessage := fmt.Sprintf("Failed to generate component pod map: %v\n", err)
		logging.LogMessage(logFile, logMessage)
		logging.Alert(w, logMessage)
		return
	}
	logging.LogComponentPodMap(componentPodMap, logFile)

	log.Printf("Node CPU frequencies successfully collected and sent.")
}

// handleGetNodeCpuHz는 /get-node-cpu-hz 요청을 처리하고 각 노드의 CPU 주파수를 응답합니다.
func handleGetNodeCpuHz(w http.ResponseWriter, promClient *prometheusClient.PrometheusClient) {
	logFile := logging.GenerateLogFile()
	defer logFile.Close()

	nodeCpuUtilizationMap, err := metricCollector.CollectNodeCpuHz(promClient)
	if err != nil {
		logMessage := fmt.Sprintf("Error collecting CPU Hz data: %v\n", err)
		logging.LogMessage(logFile, logMessage)
		logging.Alert(w, logMessage)
	}
	logging.LogNodeCpuHz(nodeCpuUtilizationMap, logFile)

	log.Printf("Node CPU frequencies successfully collected and sent.")
}

// handleGetMonitoringInfo는 /get-monitoring-info 요청을 처리하여 컴포넌트 그래프를 생성하고 모니터링 정보를 반환
func handleGetMonitoringInfo(w http.ResponseWriter, r *http.Request, promClient *prometheusClient.PrometheusClient) {
	namespace := r.URL.Query().Get("namespace")
	duration := r.URL.Query().Get("duration")

	if namespace == "" || duration == "" {
		logging.Alert(w, "fail\n")
		return
	}

	logFile := logging.GenerateLogFile()
	defer logFile.Close()

	componentGraph, err := graphGenerator.GenerateGraph(promClient, namespace)
	if err != nil {
		logMessage := fmt.Sprintf("Failed to generate graph: %v\n", err)
		logging.LogMessage(logFile, logMessage)
		logging.Alert(w, logMessage)
		return
	}
	logging.LogComponentGraph(componentGraph, logFile)

	componentPodMap, err := metricCollector.MapComponentToPodInfo(promClient, namespace)
	if err != nil {
		logMessage := fmt.Sprintf("Failed to generate component pod map: %v\n", err)
		logging.LogMessage(logFile, logMessage)
		logging.Alert(w, logMessage)
		return
	}
	logging.LogComponentPodMap(componentPodMap, logFile)

	requestCountMap, err := metricCollector.CollectRequestCountPerPod(promClient, namespace, componentGraph, componentPodMap, duration)
	if err != nil {
		logMessage := fmt.Sprintf("Error collecting request count data: %v\n", err)
		logging.LogMessage(logFile, logMessage)
		logging.Alert(w, logMessage)
		return
	}
	logging.LogRequestCountPerPod(requestCountMap, logFile)

	requestDurationMap, err := metricCollector.CollectRequestDurationPerPod(promClient, namespace, componentGraph, componentPodMap, duration, requestCountMap)
	if err != nil {
		logMessage := fmt.Sprintf("Error collecting request duration data: %v\n", err)
		logging.LogMessage(logFile, logMessage)
		logging.Alert(w, logMessage)
		return
	}
	logging.LogRequestDurationData(requestDurationMap, logFile)

	nodeCpuUtilizationMap, err := metricCollector.CollectNodeCpuHz(promClient)
	if err != nil {
		logMessage := fmt.Sprintf("Error collecting CPU frequency data: %v\n", err)
		logging.LogMessage(logFile, logMessage)
		logging.Alert(w, logMessage)
		return
	}
	logging.LogNodeCpuHz(nodeCpuUtilizationMap, logFile)
}

func handleTrafficSchedule(w http.ResponseWriter, r *http.Request, promClient *prometheusClient.PrometheusClient) {
	namespace := r.URL.Query().Get("namespace")
	duration := r.URL.Query().Get("duration")

	if namespace == "" || duration == "" {
		logging.Alert(w, "fail\n")
		return
	}

	logFile := logging.GenerateLogFile()
	defer logFile.Close()

	// 컴포넌트의 graph (node와 edge를) 구함
	componentGraph, err := graphGenerator.GenerateGraph(promClient, namespace)
	if err != nil {
		logging.Alert(w, "get graph generate fail\n")
		return
	}
	logging.LogComponentGraph(componentGraph, logFile)

	//component와 pod 매핑한 정보 구함
	componentPodMap, err := metricCollector.MapComponentToPodInfo(promClient, namespace)
	if err != nil {
		logging.Alert(w, "get component pod map fail\n")
		return
	}
	logging.LogComponentPodMap(componentPodMap, logFile)

	// 각 파드 별로 소스 파드로 부터 수신한 요청 량을 출력함
	requestCountMap, err := metricCollector.CollectRequestCountPerPod(promClient, namespace, componentGraph, componentPodMap, duration)
	if err != nil {
		logging.Alert(w, "get request count map fail\n")
		return
	}
	logging.LogRequestCountPerPod(requestCountMap, logFile)

	requestDurationMap, err := metricCollector.CollectRequestDurationPerPod(promClient, namespace, componentGraph, componentPodMap, duration, requestCountMap)
	if err != nil {
		logging.Alert(w, "get request duration map fail\n")
		return
	}
	logging.LogRequestDurationData(requestDurationMap, logFile)

	cpuUtilizationMap, err := metricCollector.CollectCpuUtilizationPerPod(promClient, namespace, componentPodMap, duration)
	if err != nil {
		logging.Alert(w, "get cpu utilization map fail\n")
		return
	}
	logging.LogCpuUtilizationPerPod(cpuUtilizationMap, logFile)

	nodeCpuHzMap, err := metricCollector.CollectNodeCpuHz(promClient)
	if err != nil {
		logging.Alert(w, "get node cpu hz map fail\n")
		return
	}
	logging.LogNodeCpuHz(nodeCpuHzMap, logFile)

	trafficAllocator.MakeEntryPoint(*componentPodMap, namespace)
	trafficAllocationResult := trafficAllocator.TrafficAllocation(namespace, componentGraph, componentPodMap)
	logging.LogTrafficAllocationResult(trafficAllocationResult, logFile)

	// 결과를 콘솔에 출력
	fmt.Printf("Request Duration Data: %+v\n", requestDurationMap)
	fmt.Printf("CPU Utilization Data: %+v\n", cpuUtilizationMap)
	fmt.Printf("CPU Hz Data: %+v\n", nodeCpuHzMap)
}
