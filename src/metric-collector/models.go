package main

// WorkerNodeData 주파수 설정
var workerFrequencies = map[string]float32{
	"worker1": 4.5,
	"worker2": 4.5,
	"worker3": 2.5,
	"worker4": 2.5,
}

// PodData represents the data for a pod
type PodData struct {
	PodName        string  `json:"pod"`
	PodIP          string  `json:"podIP"`
	Port           int32   `json:"podPort"`    // 수신 중인 포트
	CpuUtilization float64 `json:"cpuUtil"`    // Percent
	RequestCount   int     `json:"requests"`   // Received requests in the specified duration
	WorkerNode     string  `json:"worker"`     // Worker node name
	Frequency      float32 `json:"frequency"`  // Worker node frequency
	CpuRequest     int64   `json:"cpuRequest"` // 추가: CPU request in millicores
}

// ComponentData represents data for a component
type ComponentData struct {
	ComponentName string    `json:"component"`
	Pods          []PodData `json:"pods"`
}

// NamespaceData represents data for a namespace
type NamespaceData struct {
	Namespace  string          `json:"namespace"`
	Components []ComponentData `json:"components"`
}
