package main

import (
	"log"

	"traffic-scheduler/graph"

	gonumgraph "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

func main() {
	dagURL := "http://10.96.9.243:21001/dag?value=15&namespace=pair"
	metricsURL := "http://10.96.9.243:21001/metrics?value=15&namespace=pair"

	// Fetch DAG data
	links, err := graph.FetchDAGData(dagURL)
	if err != nil {
		log.Fatalf("Failed to fetch DAG data: %v", err)
	}

	// Fetch metrics data
	components, err := graph.FetchMetricsData(metricsURL)
	if err != nil {
		log.Fatalf("Failed to fetch Metrics data: %v", err)
	}

	// Create DAG
	dagGraph := graph.NewDAG(links, components)

	// Create graph
	g := simple.NewDirectedGraph()
	var nodeID int64 = 1
	nodes := make(map[string]gonumgraph.Node) // Key: "component/pod"

	// Add nodes
	graph.AddNodesToGraph(g, components, nodes, &nodeID)

	// Add edges
	graph.AddLinksToGraph(g, links, dagGraph, nodes)

	// Print graph
	graph.PrintGraph(g, nodes)

	// Run traffic allocation algorithm across the entire DAG
	//algorithm.AllocateTrafficInDAG(g, dagGraph, nodes)

	// Print graph with updated weights
	//algorithm.PrintWeightedGraph(g)

	// Additional algorithm applications can be added here
}
