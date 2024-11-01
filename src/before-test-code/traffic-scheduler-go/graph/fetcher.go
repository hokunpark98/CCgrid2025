package graph

import (
	"encoding/json"
	"net/http"
)

// FetchDAGData fetches component relationships from the DAG endpoint
func FetchDAGData(url string) ([]Link, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var links []Link
	if err := json.NewDecoder(resp.Body).Decode(&links); err != nil {
		return nil, err
	}
	return links, nil
}

// FetchMetricsData fetches component and replica information from the metrics endpoint
func FetchMetricsData(metricsURL string) ([]Component, error) {
	resp, err := http.Get(metricsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Components []Component `json:"components"`
	}
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&data); err != nil {
		return nil, err
	}

	// 각 Replica에 ComponentName 설정
	for i := range data.Components {
		component := &data.Components[i]
		for j := range component.Pods {
			component.Pods[j].ComponentName = component.ComponentName
		}
	}

	return data.Components, nil
}
