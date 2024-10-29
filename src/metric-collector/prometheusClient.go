package main

import (
	"context"
	"log"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// PrometheusClient simplifies interactions with the Prometheus API.
type PrometheusClient struct {
	api v1.API
}

// NewPrometheusClient initializes the Prometheus API client.
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

// Query sends a query to Prometheus and returns the result.
func (p *PrometheusClient) Query(query string) (model.Vector, error) {
	result, warnings, err := p.api.Query(context.Background(), query, time.Now())
	if err != nil {
		return nil, err
	}
	if len(warnings) > 0 {
		log.Printf("Prometheus Warnings: %v\n", warnings)
	}
	return result.(model.Vector), nil
}
