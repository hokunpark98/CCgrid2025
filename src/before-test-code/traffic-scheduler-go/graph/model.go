package graph

type Link struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

type Replica struct {
	ComponentName   string  `json:"component"`
	PodName         string  `json:"pod"`
	PodIP           string  `json:"podIP"`
	Port            int32   `json:"podPort"`
	CpuUtilization  float64 `json:"cpuUtil"`
	RequestCount    int     `json:"requests"`
	WorkerNode      string  `json:"worker"`
	Frequency       float32 `json:"frequency"`
	CpuRequest      int64   `json:"cpuRequest"`
	IncomingTraffic float64 // Replica's incoming traffic
	Capacity        float64 // Replica's capacity (traffic it can send or receive)
	RemainingCap    float64 // Replica's remaining capacity
}

type Component struct {
	ComponentName string    `json:"component"`
	Pods          []Replica `json:"pods"`
}

type ReplicaNode struct {
	IDValue int64
	Replica Replica
}

func (n ReplicaNode) ID() int64 {
	return n.IDValue
}

type DAG struct {
	Component map[string][]string
	Replicas  map[string][]Replica
}
