package graph

import (
	"fmt"

	gonumgraph "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

// AddLinksToGraph adds edges to the graph
func AddLinksToGraph(g *simple.DirectedGraph, links []Link, dagGraph *DAG, nodes map[string]gonumgraph.Node) {
	for _, link := range links {
		fromComponent := link.Source
		toComponent := link.Destination

		fromReplicas := dagGraph.Replicas[fromComponent]
		toReplicas := dagGraph.Replicas[toComponent]

		for _, fromReplica := range fromReplicas {
			fromKey := fmt.Sprintf("%s/%s", fromComponent, fromReplica.PodName)
			fromNode := nodes[fromKey]
			for _, toReplica := range toReplicas {
				toKey := fmt.Sprintf("%s/%s", toComponent, toReplica.PodName)
				toNode := nodes[toKey]

				edge := simple.Edge{F: fromNode, T: toNode}
				g.SetEdge(edge)
			}
		}
	}
}
