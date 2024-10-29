package main

import (
	"fmt"
	"strconv"
	"strings"
)

// PodStats 구조체 정의
type PodStats struct {
	PodName    string
	Bandwidth  int64
	CPUUtil    float64
	WorkerName string
}

// 제외할 파드 이름 접두사
var excludePrefixes = []string{"kube-proxy", "blackbox-exporter", "calico", "istiod", "metric-probe", "grafana"}

// collector에게 전송할 정보를 필터링 하는 부분에 대한 코드
func parsePodStats(stats *KubeletStats) map[string]PodStats {
	podStatsMap := make(map[string]PodStats)
	workerName := stats.Node.NodeName // nodeName을 workerName으로 사용

	for _, pod := range stats.Pods {
		podName := pod.PodRef.Name

		// 제외할 파드 이름 필터링
		shouldExclude := false
		for _, prefix := range excludePrefixes {
			if strings.HasPrefix(podName, prefix) {
				shouldExclude = true
				break
			}
		}
		if shouldExclude {
			continue
		}

		// Bandwidth 계산 (eth0)
		var bandwidth int64
		for _, iface := range pod.Network.Interfaces {
			if iface.Name == "eth0" {
				bandwidth = iface.RxBytes + iface.TxBytes
				break
			}
		}

		// 나노코어를 밀리코어로 변환하고 소수점 1자리까지 반올림
		cpuUtilMillicores := float64(pod.CPU.UsageNanoCores) / 1e6
		cpuUtilRounded, _ := strconv.ParseFloat(fmt.Sprintf("%.1f", cpuUtilMillicores), 64)

		// PodStats 맵에 추가
		podStatsMap[podName] = PodStats{
			PodName:    podName,
			Bandwidth:  bandwidth,
			CPUUtil:    cpuUtilRounded,
			WorkerName: workerName,
		}
	}

	return podStatsMap
}
