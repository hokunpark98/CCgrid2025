package main

// WorkerNodeData 주파수 설정
var workerFrequencies = map[string]float32{
	"worker1": 4.5,
	"worker2": 4.5,
	"worker3": 2.5,
	"worker4": 2.5,
	"worker5": 3,
}

// ReplicaData represents the data for a Replica
type ReplicaData struct {
	ReplicaName    string  `json:"Replica"`
	ReplicaVersion string  `json:"ReplicaVersion"`
	IP             string  `json:"IP"`
	Port           int32   `json:"Port"`       // 수신 중인 포트
	CpuUtil        float64 `json:"CpuUtil"`    // Percent
	Requests       int     `json:"Requests"`   // Received requests in the specified duration
	Worker         string  `json:"Worker"`     // Worker node name
	Frequency      float32 `json:"Frequency"`  // Worker node frequency
	CpuRequest     int64   `json:"CpuRequest"` // 추가: CPU request in millicores
}

// ComponentData represents data for a component
type ComponentData struct {
	ComponentName string        `json:"Component"`
	Replicas      []ReplicaData `json:"Replicas"`
}

// NamespaceData represents data for a namespace
type NamespaceData struct {
	Namespace  string          `json:"Namespace"`
	Components []ComponentData `json:"Components"`
}

// 의존성 그래프를 표현하기 위한 구조체
type DependencyData struct {
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
}
