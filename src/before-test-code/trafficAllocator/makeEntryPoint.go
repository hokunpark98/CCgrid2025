package trafficAllocator

import (
	"fmt"
	"log"
	"os"
	"text/template"
	"traffic-scheduler/metricCollector"
)

// YAML 템플릿 정의
const yamlTemplate = `
apiVersion: networking.istio.io/v1alpha3
kind: ServiceEntry
metadata:
  name: {{.PodName}}-{{.Port}}
  namespace: {{.Namespace}}
spec:
  hosts:
  - {{.PodName}}
  addresses:
  - {{.PodIP}}
  ports:
  - number: {{.Port}}
    name: http
    protocol: HTTP
  resolution: STATIC
  location: MESH_INTERNAL
  endpoints:
  - address: {{.PodIP}}
    ports:
      http: {{.Port}}
`

// writeServiceEntryYAML 함수는 주어진 componentPodMap에서 각 pod와 포트를 바탕으로 YAML 파일을 생성합니다.
func MakeEntryPoint(componentPodMap metricCollector.ComponentPodMap, namespace string) error {
	// 모든 컴포넌트에 대해 반복
	for component, pods := range componentPodMap.Components {
		// 각 컴포넌트의 파드에 대해 반복
		for _, podInfo := range pods {
			// 각 포트에 대해 YAML 파일 생성
			for _, port := range podInfo.Ports {
				// 파일 이름을 "component명/pod명:port.yaml"로 설정
				filename := fmt.Sprintf("etc/entryPointYamls/%s/%s/%s-%s.yaml", namespace, component, podInfo.PodName, port)
				folderPath := fmt.Sprintf("etc/entryPointYamls/%s/%s", namespace, component)
				// 디렉터리 생성 (없으면)
				err := os.MkdirAll(folderPath, os.ModePerm)
				if err != nil {
					log.Fatalf("Error creating directory: %v", err)
					return err
				}

				// 템플릿 데이터를 채우기 위한 구조체
				data := struct {
					PodName   string
					PodIP     string
					Namespace string
					Port      string
				}{
					PodName:   podInfo.PodName,
					PodIP:     podInfo.PodIP,
					Namespace: namespace,
					Port:      port,
				}

				// YAML 템플릿 파싱
				tmpl, err := template.New("serviceEntry").Parse(yamlTemplate)
				if err != nil {
					log.Fatalf("Error parsing template: %v", err)
					return err
				}

				// 파일 생성
				file, err := os.Create(filename)
				if err != nil {
					log.Fatalf("Error creating file: %v", err)
					return err
				}
				defer file.Close()

				// 템플릿 데이터를 파일에 쓰기
				err = tmpl.Execute(file, data)
				if err != nil {
					log.Fatalf("Error executing template: %v", err)
					return err
				}
				fmt.Printf("Generated %s\n", filename)
			}
		}
	}

	return nil
}
