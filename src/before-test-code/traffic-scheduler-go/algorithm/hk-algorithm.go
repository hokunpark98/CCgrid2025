package algorithm

import (
	"fmt"
	"sort"

	"traffic-scheduler/graph"

	gonumgraph "gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/simple"
)

// WeightedEdge represents an edge with a weight
type WeightedEdge struct {
	simple.Edge
	Weight float64
}

// WeightedGraph is a directed graph with weighted edges
type WeightedGraph struct {
	Nodes map[int64]gonumgraph.Node
	Edges map[gonumgraph.Node][]WeightedEdge
}

// NewWeightedGraph creates and initializes a WeightedGraph
func NewWeightedGraph() *WeightedGraph {
	return &WeightedGraph{
		Nodes: make(map[int64]gonumgraph.Node),
		Edges: make(map[gonumgraph.Node][]WeightedEdge),
	}
}

// AddNode adds a node to the graph
func (g *WeightedGraph) AddNode(node gonumgraph.Node) {
	g.Nodes[node.ID()] = node
}

// SetEdge adds an edge with a weight to the graph
func (g *WeightedGraph) SetEdge(edge WeightedEdge) {
	g.Edges[edge.From()] = append(g.Edges[edge.From()], edge)
}

// TopologicalSort performs a topological sort on the DAG and returns the ordered component names
func TopologicalSort(dagGraph *graph.DAG) []string {
	visited := make(map[string]bool)
	stack := []string{}

	// Helper function for recursive DFS
	var dfs func(string)
	dfs = func(node string) {
		visited[node] = true
		for _, child := range dagGraph.Component[node] {
			if !visited[child] {
				dfs(child)
			}
		}
		stack = append(stack, node)
	}

	// Apply DFS on all components to ensure we cover disconnected parts
	for component := range dagGraph.Component {
		if !visited[component] {
			dfs(component)
		}
	}

	// Reverse the stack to get the topological order
	// as DFS-based topological sort will give reverse order in stack
	orderedComponents := make([]string, len(stack))
	for i := len(stack) - 1; i >= 0; i-- {
		orderedComponents[len(stack)-1-i] = stack[i]
	}

	return orderedComponents
}

// AllocateTrafficInDAG allocates traffic across the entire DAG
func AllocateTrafficInDAG(g *WeightedGraph, dagGraph *graph.DAG, nodes map[string]gonumgraph.Node) {
	// 1. Initialize capacities based on Frequency
	initializeCapacities(dagGraph)

	// 2. Get topological ordering of components
	orderedComponents := TopologicalSort(dagGraph)

	// 3. Distribute incoming traffic to the top-level component's replicas
	totalTraffic := 100.0 // Assuming total incoming traffic is 100 units
	topComponent := orderedComponents[0]
	distributeTrafficToTopComponent(dagGraph, topComponent, totalTraffic)

	// 4. Allocate traffic between components
	for i := 0; i < len(orderedComponents)-1; i++ {
		uc := orderedComponents[i]
		dc := orderedComponents[i+1]
		allocateBetweenComponents(dagGraph, uc, dc, g, nodes)
	}
}

// Initialize capacities of replicas based on their Frequency
func initializeCapacities(dagGraph *graph.DAG) {
	for _, replicas := range dagGraph.Replicas {
		var totalFrequency float32
		for _, replica := range replicas {
			totalFrequency += replica.Frequency
		}
		for i, replica := range replicas {
			capacity := float64(replica.Frequency) / float64(totalFrequency)
			dagGraph.Replicas[replica.ComponentName][i].Capacity = capacity
			dagGraph.Replicas[replica.ComponentName][i].RemainingCap = capacity
		}
	}
}

// Distribute traffic to the top-level component's replicas
func distributeTrafficToTopComponent(dagGraph *graph.DAG, componentName string, totalTraffic float64) {
	replicas := dagGraph.Replicas[componentName]
	var totalCapacity float64
	for _, replica := range replicas {
		totalCapacity += replica.Capacity
	}

	var allocatedTraffic float64
	for i, replica := range replicas {
		proportion := replica.Capacity / totalCapacity
		traffic := proportion * totalTraffic
		// Round to the nearest integer for integer allocation
		traffic = float64(int(traffic + 0.5))
		dagGraph.Replicas[componentName][i].IncomingTraffic = traffic
		allocatedTraffic += traffic
	}

	// Adjust for any rounding errors to ensure total traffic is conserved
	roundingError := totalTraffic - allocatedTraffic
	if roundingError != 0 {
		dagGraph.Replicas[componentName][0].IncomingTraffic += roundingError
	}
}

// Allocate traffic between two components
func allocateBetweenComponents(dagGraph *graph.DAG, ucName, dcName string, g *WeightedGraph, nodes map[string]gonumgraph.Node) {
	uReplicas := dagGraph.Replicas[ucName]
	dReplicas := dagGraph.Replicas[dcName]

	// Initialize remaining capacities
	for i := range uReplicas {
		uReplicas[i].RemainingCap = uReplicas[i].IncomingTraffic
	}
	for i := range dReplicas {
		dReplicas[i].RemainingCap = dReplicas[i].Capacity
	}

	// Edge Full Allocation
	for i := range uReplicas {
		for j := range dReplicas {
			allocateTrafficEdge(&uReplicas[i], &dReplicas[j], g, nodes)
		}
	}

	// Deficient Edge Minimization Allocation
	deficientEdges := make(map[int]int) // Key: index of DC replica, Value: number of deficient edges
	for {
		allAllocated := true
		for i := range uReplicas {
			if uReplicas[i].RemainingCap > 0 {
				allAllocated = false
				dcIndex := selectDCWithMinDeficientEdges(dReplicas, deficientEdges)
				allocated := allocateTrafficEdge(&uReplicas[i], &dReplicas[dcIndex], g, nodes)
				if !allocated {
					deficientEdges[dcIndex]++
				}
			}
		}
		if allAllocated {
			break
		}
	}
}

// Allocate traffic between a pair of replicas
func allocateTrafficEdge(uReplica, dReplica *graph.Replica, g *WeightedGraph, nodes map[string]gonumgraph.Node) bool {
	minCap := min(uReplica.RemainingCap, dReplica.RemainingCap)
	if minCap <= 0 {
		return false
	}

	keyFrom := fmt.Sprintf("%s/%s", uReplica.ComponentName, uReplica.PodName)
	keyTo := fmt.Sprintf("%s/%s", dReplica.ComponentName, dReplica.PodName)
	fromNode := nodes[keyFrom]
	toNode := nodes[keyTo]

	// Create and add a weighted edge
	weightedEdge := WeightedEdge{
		Edge:   simple.Edge{F: fromNode, T: toNode},
		Weight: minCap,
	}
	g.SetEdge(weightedEdge)

	// Update remaining capacities
	uReplica.RemainingCap -= minCap
	dReplica.RemainingCap -= minCap

	return true
}

// Select DC replica with minimum deficient edges
func selectDCWithMinDeficientEdges(dReplicas []graph.Replica, deficientEdges map[int]int) int {
	type dcInfo struct {
		Index         int
		DeficientEdge int
		RemainingCap  float64
	}
	var dcList []dcInfo
	for i, dReplica := range dReplicas {
		dcList = append(dcList, dcInfo{
			Index:         i,
			DeficientEdge: deficientEdges[i],
			RemainingCap:  dReplica.RemainingCap,
		})
	}

	// Sort by deficient edges, then by remaining capacity
	sort.Slice(dcList, func(i, j int) bool {
		if dcList[i].DeficientEdge == dcList[j].DeficientEdge {
			return dcList[i].RemainingCap > dcList[j].RemainingCap
		}
		return dcList[i].DeficientEdge < dcList[j].DeficientEdge
	})

	return dcList[0].Index
}

// Utility functions
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// PrintWeightedGraph prints edges with their weights
// PrintWeightedGraph prints edges with their weights
func PrintWeightedGraph(g *WeightedGraph) {
	fmt.Println("Graph Edges with Weights:")
	for fromNode, edges := range g.Edges {
		for _, edge := range edges {
			fromReplica, fromOk := fromNode.(graph.ReplicaNode)
			toReplica, toOk := edge.To().(graph.ReplicaNode)
			if fromOk && toOk {
				fmt.Printf("From: %s -> To: %s, Weight: %.2f\n", fromReplica.Replica.PodName, toReplica.Replica.PodName, edge.Weight)
			} else {
				fmt.Println("Error: Node type mismatch. Ensure nodes are of type graph.ReplicaNode.")
			}
		}
	}
}
