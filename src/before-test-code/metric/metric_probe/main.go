package main

import (
	"encoding/json"
	"fmt"
	"time"
)

func main() {
	// 수집 및 전송 주기 설정
	url := "https://localhost:10250/stats/summary"
	//targetURL := "http://metric-collector:15000/metrics"
	targetURL := "http://192.168.0.10:15000/probe"

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 각 주기마다 비동기로 고루틴 생성
		go func() {
			// 데이터 가져오기
			stats, err := fetchStats(url)
			if err != nil {
				fmt.Printf("Error fetching stats: %v\n", err)
				return
			}

			// PodStats 맵 생성
			podStatsMap := parsePodStats(stats)

			// podStatsMap을 JSON 형식으로 로그 출력
			podStatsJSON, err := json.MarshalIndent(podStatsMap, "", "  ")
			if err != nil {
				fmt.Printf("Error marshalling podStatsMap: %v\n", err)
			} else {
				fmt.Printf("PodStatsMap: %s\n", string(podStatsJSON))
			}

			// PodStats 비동기로 전송
			go func() {
				if err := sendStats(targetURL, podStatsMap); err != nil {
					fmt.Printf("Error sending stats: %v\n", err)
				} else {
					fmt.Println("Stats successfully sent")
				}
			}()
		}()
	}
}
