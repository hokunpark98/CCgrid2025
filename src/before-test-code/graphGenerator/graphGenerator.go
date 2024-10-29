package graphGenerator

import (
	"fmt"

	"traffic-scheduler/prometheusClient"

	"gonum.org/v1/gonum/graph/simple"
)

// ComponentGraph 구조체 정의
type ComponentGraph struct {
	Components map[string][]string `json:"components"`
}

// GenerateGraph 함수: 네임스페이스에 대한 컴포넌트 그래프 생성
func GenerateGraph(promClient *prometheusClient.PrometheusClient, namespace string) (*ComponentGraph, error) {
	query := fmt.Sprintf(`istio_requests_total{kubernetes_namespace="%s"}`, namespace)

	result, err := promClient.Query(query)
	if err != nil {
		return nil, err
	}

	graph := simple.NewDirectedGraph()
	nodeMap := make(map[string]int64)
	nodeNames := make(map[int64]string)

	// 각 샘플을 순회하며 그래프 노드 및 엣지 생성
	for _, sample := range result {
		source := string(sample.Metric["source_app"])
		dest := string(sample.Metric["destination_app"])

		if source == "unknown" || dest == "unknown" {
			continue
		}

		// UC 추가
		if _, exists := nodeMap[source]; !exists {
			node := graph.NewNode()
			nodeMap[source] = node.ID()
			nodeNames[node.ID()] = source
			graph.AddNode(node)
		}

		// DC 추가
		if _, exists := nodeMap[dest]; !exists {
			node := graph.NewNode()
			nodeMap[dest] = node.ID()
			nodeNames[node.ID()] = dest
			graph.AddNode(node)
		}

		// 링크 추가
		graph.SetEdge(graph.NewEdge(graph.Node(nodeMap[source]), graph.Node(nodeMap[dest])))
	}

	// 컴포넌트 맵 생성 및 링크 추가
	components := make(map[string][]string)

	it := graph.Edges()
	for it.Next() {
		edge := it.Edge()
		fromName := nodeNames[edge.From().ID()]
		toName := nodeNames[edge.To().ID()]

		// 컴포넌트가 맵에 없으면 새로 생성 후 링크 추가
		components[fromName] = append(components[fromName], toName)
	}

	return &ComponentGraph{
		Components: components,
	}, nil
}
