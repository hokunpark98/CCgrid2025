package graph

import (
	"fmt"

	gonumgraph "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

// PrintGraph prints the nodes and edges of the graph
func PrintGraph(g *simple.DirectedGraph, nodes map[string]gonumgraph.Node) {
	fmt.Println("Nodes:")
	for _, node := range nodes {
		replicaNode := node.(ReplicaNode)
		fmt.Printf("ID: %d, Replica: %s\n", replicaNode.ID(), replicaNode.Replica.PodName)
	}

	fmt.Println("\nLinks:")
	linksIter := g.Edges()
	for linksIter.Next() {
		link := linksIter.Edge().(simple.Edge)
		from := link.From().(ReplicaNode)
		to := link.To().(ReplicaNode)
		fmt.Printf("From: %s -> To: %s\n", from.Replica.PodName, to.Replica.PodName)
	}
}
