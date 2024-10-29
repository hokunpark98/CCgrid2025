package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// sendStats 함수: PodStats 맵을 HTTP POST 요청으로 전송
func sendStats(url string, stats map[string]PodStats) error {
	data, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send stats, status code: %d", resp.StatusCode)
	}

	return nil
}
