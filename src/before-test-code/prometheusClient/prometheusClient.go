package prometheusClient

import (
	"context"
	"log"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// PrometheusClient 구조체는 Prometheus API와의 상호작용을 단순화합니다.
type PrometheusClient struct {
	api v1.API
}

// Prometheus API 클라이언트 초기화
func NewPrometheusClient(address string) (*PrometheusClient, error) {
	client, err := api.NewClient(api.Config{
		Address: address,
	})
	if err != nil {
		return nil, err
	}
	return &PrometheusClient{
		api: v1.NewAPI(client),
	}, nil
}

// Query는 Prometheus에 쿼리를 전송하고 결과 반환
func (p *PrometheusClient) Query(query string) (model.Vector, error) {
	result, warnings, err := p.api.Query(context.Background(), query, time.Now())
	if err != nil {
		return nil, err
	}
	if len(warnings) > 0 {
		log.Printf("Warnings: %v\n", warnings)
	}
	return result.(model.Vector), nil
}
