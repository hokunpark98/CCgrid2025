package graph

import (
	"fmt"

	gonumgraph "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

// AddNodesToGraph adds nodes to the graph
func AddNodesToGraph(g *simple.DirectedGraph, components []Component, nodes map[string]gonumgraph.Node, nodeID *int64) {
	for _, component := range components {
		// component.ComponentName이 비어 있는지 확인
		if component.ComponentName == "" {
			fmt.Println("Warning: ComponentName is empty for a component.")
		}
		for _, replica := range component.Pods {
			// replica.ComponentName이 비어 있으면 설정
			if replica.ComponentName == "" {
				replica.ComponentName = component.ComponentName
			}

			node := ReplicaNode{
				IDValue: *nodeID,
				Replica: replica,
			}
			g.AddNode(node)
			key := fmt.Sprintf("%s/%s", replica.ComponentName, replica.PodName)
			nodes[key] = node
			*nodeID++
		}
	}
}
