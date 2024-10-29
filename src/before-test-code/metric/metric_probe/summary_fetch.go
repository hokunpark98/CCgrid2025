package main

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"
)

// KubeletStats 구조체 정의
type KubeletStats struct {
	Node struct {
		NodeName string `json:"nodeName"`
	} `json:"node"`
	Pods []struct {
		PodRef struct {
			Name string `json:"name"`
		} `json:"podRef"`
		Network struct {
			Interfaces []struct {
				Name    string `json:"name"`
				RxBytes int64  `json:"rxBytes"`
				TxBytes int64  `json:"txBytes"`
			} `json:"interfaces"`
		} `json:"network"`
		CPU struct {
			UsageNanoCores int64 `json:"usageNanoCores"`
		} `json:"cpu"`
	} `json:"pods"`
}

// fetchStats 함수
func fetchStats(url string) (*KubeletStats, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var stats KubeletStats
	err = json.Unmarshal(data, &stats)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}
