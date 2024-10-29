package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	// /probe 및 /metrics 엔드포인트 핸들러 설정
	http.HandleFunc("/probe", probeHandler)
	http.HandleFunc("/metrics", metricsHandler)

	// 서버 시작
	port := "15000"
	fmt.Printf("Starting server on port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
