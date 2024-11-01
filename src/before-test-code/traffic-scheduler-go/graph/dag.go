package graph

import "fmt"

func NewDAG(links []Link, components []Component) *DAG {
	dag := &DAG{
		Component: make(map[string][]string),
		Replicas:  make(map[string][]Replica),
	}

	for _, link := range links {
		dag.Component[link.Source] = append(dag.Component[link.Source], link.Destination)
	}

	for _, component := range components {
		dag.Replicas[component.ComponentName] = component.Pods
	}
	return dag
}

func (d *DAG) Print() {
	fmt.Println("DAG Structure with Replicas:")
	for node, children := range d.Component {
		fmt.Printf("%s -> %v\n", node, children)
		if replicas, exists := d.Replicas[node]; exists {
			for _, replica := range replicas {
				fmt.Printf("  Replica: %s, IP: %s, Port: %d, CPU Util: %.4f, Worker: %s\n",
					replica.PodName, replica.PodIP, replica.Port, replica.CpuUtilization, replica.WorkerNode)
			}
		}
	}
}
