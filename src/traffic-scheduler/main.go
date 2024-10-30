package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/gonum/graph/simple"
	"github.com/gonum/graph/topo"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// 서비스 의존성 구조체 정의
type ServiceDependency struct {
	Source      string
	Destination string
}

// Prometheus 쿼리를 통해 Istio 트래픽 데이터 수집
func getServiceDependencies(client v1.API, namespace string) ([]ServiceDependency, error) {
	// istio_requests_total 메트릭을 쿼리하여 서비스 간 호출 관계를 수집
	query := fmt.Sprintf(`istio_requests_total{reporter="source", source_workload_namespace="%s"}`, namespace)
	result, warnings, err := client.Query(context.Background(), query, time.Now())
	if err != nil {
		return nil, err
	}
	if len(warnings) > 0 {
		log.Printf("Warnings: %v\n", warnings)
	}

	dependencies := []ServiceDependency{}
	for _, sample := range result.(model.Vector) {
		source := string(sample.Metric["source_workload"])
		destination := string(sample.Metric["destination_workload"])
		dependencies = append(dependencies, ServiceDependency{Source: source, Destination: destination})
	}

	return dependencies, nil
}

// DAG 생성
func createDAG(dependencies []ServiceDependency) *simple.DirectedGraph {
	dag := simple.NewDirectedGraph()
	nodeMap := make(map[string]simple.Node)

	for _, dep := range dependencies {
		if _, exists := nodeMap[dep.Source]; !exists {
			nodeMap[dep.Source] = dag.NewNode()
			dag.AddNode(nodeMap[dep.Source])
		}
		if _, exists := nodeMap[dep.Destination]; !exists {
			nodeMap[dep.Destination] = dag.NewNode()
			dag.AddNode(nodeMap[dep.Destination])
		}
		dag.SetEdge(dag.NewEdge(nodeMap[dep.Source], nodeMap[dep.Destination]))
	}

	return dag
}

func main() {
	// Prometheus API 클라이언트 설정
	client, err := api.NewClient(api.Config{
		Address: "http://prometheus-server:9090",
	})
	if err != nil {
		log.Fatalf("Error creating Prometheus client: %v", err)
	}
	v1api := v1.NewAPI(client)

	namespace := "your-namespace" // 네임스페이스 설정
	dependencies, err := getServiceDependencies(v1api, namespace)
	if err != nil {
		log.Fatalf("Error fetching dependencies: %v", err)
	}

	dag := createDAG(dependencies)

	// DAG의 위상 정렬 결과 출력
	sorted, err := topo.Sort(dag)
	if err != nil {
		log.Fatalf("Error sorting DAG: %v", err)
	}
	fmt.Println("Service Dependency DAG (Topologically Sorted):")
	for _, node := range sorted {
		fmt.Println(node.ID())
	}

	// 각 간선 출력
	fmt.Println("\nService Dependencies:")
	for _, dep := range dependencies {
		fmt.Printf("%s -> %s\n", dep.Source, dep.Destination)
	}
}
