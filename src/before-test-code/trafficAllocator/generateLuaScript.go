package trafficAllocator

import (
	"fmt"
	"log"
	"os"
	"strings"
	"traffic-scheduler/metricCollector"
)

// GenerateLuaScript generates Lua script for traffic allocation based on the proportion map.
func GenerateLuaScript(namespace string, proportionMap *ProportionMap, componentPodMap *metricCollector.ComponentPodMap) {
	for sourceComponent, destMap := range proportionMap.Components {
		var sb strings.Builder

		// EnvoyFilter 헤더와 메타데이터 부분 작성
		sb.WriteString(fmt.Sprintf(
			"apiVersion: networking.istio.io/v1alpha3\n"+
				"kind: EnvoyFilter\n"+
				"metadata:\n"+
				"  name: %s-filter\n"+
				"  namespace: %s\n"+
				"spec:\n"+
				"  workloadSelector:\n"+
				"    labels:\n"+
				"      app: %s\n"+
				"  configPatches:\n"+
				"  - applyTo: HTTP_FILTER\n"+
				"    match:\n"+
				"      context: SIDECAR_OUTBOUND\n"+
				"      listener:\n"+
				"        filterChain:\n"+
				"          filter:\n"+
				"            name: envoy.filters.network.http_connection_manager\n"+
				"    patch:\n"+
				"      operation: INSERT_BEFORE\n"+
				"      value:\n"+
				"        name: envoy.filters.http.lua\n"+
				"        typed_config:\n"+
				"          \"@type\": type.googleapis.com/envoy.extensions.filters.http.lua.v3.Lua\n"+
				"          inline_code: |\n"+
				"            local pod_ip = nil\n\n"+
				"            function envoy_on_request(request_handle)\n"+
				"              -- Pod IP를 한 번만 가져오기\n"+
				"              if not pod_ip then\n"+
				"                local handle = io.popen(\"hostname -i\")\n"+
				"                pod_ip = handle:read(\"*a\"):match(\"^%%s*(.-)%%s*$\")\n"+
				"                handle:close()\n"+
				"              end\n\n"+
				"              -- 요청에서 목적지 호스트(:authority) 헤더 추출\n"+
				"              local destination = request_handle:headers():get(\":authority\")\n"+
				"              local domain = destination:match(\"^([^:]+)\")\n\n",
			sourceComponent, namespace, sourceComponent))

		// 각 목적지에 대해 루프
		for destinationComponent, sourcePodDataList := range destMap {
			sb.WriteString(fmt.Sprintf("              if domain == \"%s\" then\n", destinationComponent))
			sb.WriteString("                local new_destination = nil\n")
			sb.WriteString("                local rand = math.random(0, 100)\n")
			// 각 소스 Pod에 대해 처리
			for i, sourcePodData := range sourcePodDataList {
				if i == 0 {
					sb.WriteString(fmt.Sprintf("                if pod_ip == \"%s\" then\n", componentPodMap.Pods[sourcePodData.SourcePodName].PodIP))
				} else {
					sb.WriteString(fmt.Sprintf("                elseif pod_ip == \"%s\" then\n", componentPodMap.Pods[sourcePodData.SourcePodName].PodIP))
				}

				// 트래픽 비율에 따른 목적지 IP 설정
				for j, proportionData := range sourcePodData.ProportionDatas {
					if j == 0 {
						sb.WriteString(fmt.Sprintf("                  if rand <= %d then\n                        new_destination = \"%s\"\n", proportionData.Proportion, componentPodMap.Pods[proportionData.DestinationPodName].PodName))
					} else {
						sb.WriteString(fmt.Sprintf("                  elseif rand <= %d then\n                        new_destination = \"%s\"\n", proportionData.Proportion, componentPodMap.Pods[proportionData.DestinationPodName].PodName))
					}
				}
				sb.WriteString("                  end\n")
			}

			sb.WriteString("                end\n")

		}

		// :authority 및 Host 헤더 수정
		sb.WriteString("                if new_destination then\n")
		sb.WriteString("                  local new_destination = new_destination .. destination:match(\"(:.*)$\")\n")
		sb.WriteString("                  request_handle:headers():replace(\":authority\", new_destination)\n")
		sb.WriteString("                  request_handle:headers():replace(\"Host\", new_destination)\n")
		sb.WriteString("                end\n")
		sb.WriteString("              end\n")
		sb.WriteString("            end\n")
		// 파일 경로 생성
		filename := fmt.Sprintf("etc/envoyFilterYamls/%s/%s.yaml", namespace, sourceComponent)
		folderPath := fmt.Sprintf("etc/envoyFilterYamls/%s", namespace)

		// 디렉터리 생성 (없으면)
		err := os.MkdirAll(folderPath, os.ModePerm)
		if err != nil {
			log.Fatalf("Error creating directory: %v", err)
		}

		// 파일 쓰기
		err = os.WriteFile(filename, []byte(sb.String()), 0644)
		if err != nil {
			fmt.Printf("Error writing file %s: %v\n", filename, err)
		} else {
			fmt.Printf("File %s successfully written.\n", filename)
		}
	}
}
