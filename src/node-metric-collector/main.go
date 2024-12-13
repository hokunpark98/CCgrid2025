package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type NodeMetrics struct {
	NodeName       string `json:"NodeName"`
	CPUTotal       string `json:"CPUTotal"`
	CPURemain      string `json:"CPURemain"`
	MemoryTotal    string `json:"MemoryTotal"`
	MemoryRemain   string `json:"MemoryRemain"`
	BandwidthTotal string `json:"BandwidthTotal"`
}

type PrometheusQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func main() {
	var kubeconfig *string
	if home := homeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			log.Fatalf("Error building kubeconfig: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating kubernetes client: %v", err)
	}

	prometheusURL := os.Getenv("PROMETHEUS_URL")
	if prometheusURL == "" {
		// 필요에 따라 수정
		prometheusURL = "http://prometheus-service.monitoring.svc.cluster.local:8080"
	}

	http.HandleFunc("/metric", func(w http.ResponseWriter, r *http.Request) {
		value := r.URL.Query().Get("value")
		if value == "" {
			value = "5" // 디폴트 5분
		}

		interval, err := strconv.Atoi(value)
		if err != nil {
			interval = 5
		}
		queryInterval := fmt.Sprintf("%dm", interval)

		nodeList, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get nodes: %v", err), http.StatusInternalServerError)
			return
		}

		var results []NodeMetrics
		for _, node := range nodeList.Items {
			cap := node.Status.Capacity

			// CPU/Memory Total 구하기
			cpuTotalQuantity := cap.Cpu()    // CPU cores
			memTotalQuantity := cap.Memory() // Bytes 단위
			cpuTotalVal := cpuTotalQuantity.AsApproximateFloat64()
			memTotalValBytes := float64(memTotalQuantity.Value()) // bytes
			memTotalGB := memTotalValBytes / (1024.0 * 1024.0 * 1024.0)

			// 노드 InternalIP
			nodeInternalIP := getNodeInternalIP(node)
			if nodeInternalIP == "" {
				log.Printf("Warning: Could not find InternalIP for node %s", node.Name)
				nodeInternalIP = "unknown"
			}
			instance := fmt.Sprintf("%s:9100", nodeInternalIP)

			// 파드 request 합계 계산
			// 해당 노드에 스케줄된 파드를 가져옴
			pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
				FieldSelector: fmt.Sprintf("spec.nodeName=%s", node.Name),
			})
			if err != nil {
				log.Printf("Failed to list pods for node %s: %v", node.Name, err)
				continue
			}

			var totalCPURequests float64    // cores
			var totalMemRequestsBytes int64 // bytes

			for _, pod := range pods.Items {
				for _, c := range pod.Spec.Containers {
					req := c.Resources.Requests
					if cpuQty, ok := req[v1.ResourceCPU]; ok {
						totalCPURequests += cpuQty.AsApproximateFloat64() // cores 단위
					}
					if memQty, ok := req[v1.ResourceMemory]; ok {
						totalMemRequestsBytes += memQty.Value() // bytes
					}
				}
			}

			// CPU Remain = CPU Total - sum of pod CPU requests
			cpuRemainVal := cpuTotalVal - totalCPURequests
			if cpuRemainVal < 0 {
				cpuRemainVal = 0
			}

			// Memory Remain (GB) = Memory Total(GB) - sum of pod requests(GB)
			totalMemRequestsGB := float64(totalMemRequestsBytes) / (1024.0 * 1024.0 * 1024.0)
			memRemainVal := memTotalGB - totalMemRequestsGB
			if memRemainVal < 0 {
				memRemainVal = 0
			}

			// Bandwidth total 계산 (기존 로직)
			bwQuery := fmt.Sprintf(`sum by (instance) (rate(node_network_receive_bytes_total{instance="%s",device!~"^lo$"}[%s])) 
+ sum by (instance) (rate(node_network_transmit_bytes_total{instance="%s",device!~"^lo$"}[%s]))`, instance, queryInterval, instance, queryInterval)
			bandwidthTotalStr, err := queryPrometheus(prometheusURL, bwQuery)
			if err != nil {
				log.Printf("Failed to query bandwidth for node %s: %v", node.Name, err)
				bandwidthTotalStr = "0"
			}
			bandwidthTotalVal, _ := strconv.ParseFloat(bandwidthTotalStr, 64)
			bandwidthTotalMbps := (bandwidthTotalVal * 8) / 1000000.0

			// Bandwidth 정수화 (floor)
			bandwidthTotalInt := int64(math.Floor(bandwidthTotalMbps))

			nodeMetrics := NodeMetrics{
				NodeName: node.Name,
				// CPU, Memory 모두 소수점 첫째 자리까지 float
				CPUTotal:       fmt.Sprintf("%.1f", cpuTotalVal),
				CPURemain:      fmt.Sprintf("%.1f", cpuRemainVal),
				MemoryTotal:    fmt.Sprintf("%.1f", memTotalGB),
				MemoryRemain:   fmt.Sprintf("%.1f", memRemainVal),
				BandwidthTotal: strconv.FormatInt(bandwidthTotalInt, 10),
			}

			results = append(results, nodeMetrics)
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(results); err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode metrics: %v", err), http.StatusInternalServerError)
			return
		}
	})

	log.Println("Starting server on :21002")
	if err := http.ListenAndServe(":21002", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func getNodeInternalIP(node v1.Node) string {
	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}

func queryPrometheus(promURL, query string) (string, error) {
	endpoint := fmt.Sprintf("%s/api/v1/query", strings.TrimSuffix(promURL, "/"))
	resp, err := http.Get(endpoint + "?query=" + url.QueryEscape(query))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 response from prometheus: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var pResp PrometheusQueryResponse
	if err := json.Unmarshal(body, &pResp); err != nil {
		return "", err
	}

	if pResp.Status != "success" || len(pResp.Data.Result) == 0 {
		return "0", nil
	}

	if len(pResp.Data.Result[0].Value) < 2 {
		return "0", nil
	}

	valStr, ok := pResp.Data.Result[0].Value[1].(string)
	if !ok {
		return "0", nil
	}

	return valStr, nil
}
