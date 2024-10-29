package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

var excludeKeywords = []string{"istio-system", "kube-system", "monitoring", "kube-proxy", "blackbox-exporter", "calico", "istiod", "metric-probe", "grafana"}
var logOffsets = make(map[string]int64) // 각 로그 파일의 마지막 오프셋 기록

func shouldExclude(podFullName string) bool {
	for _, keyword := range excludeKeywords {
		if strings.Contains(podFullName, keyword) {
			return true
		}
	}
	return false
}

func getFilteredLogURLs() ([]string, map[string]string, error) {
	cmd := exec.Command("curl", "-k", "-L", "https://192.168.0.11:10250/logs/pods")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, fmt.Errorf("명령어 실행 오류: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, fmt.Errorf("명령어 시작 오류: %w", err)
	}

	var logURLs []string
	podNamespaceMap := make(map[string]string)
	scanner := bufio.NewScanner(stdout)
	urlPattern := regexp.MustCompile(`href="([^"]+)"`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := urlPattern.FindStringSubmatch(line)
		if len(matches) > 1 && !shouldExclude(matches[1]) {
			fullPath := "https://192.168.0.11:10250/logs/pods/" + matches[1] + "istio-proxy/0.log"
			logURLs = append(logURLs, fullPath)
			parts := strings.Split(matches[1], "_")
			if len(parts) >= 2 {
				namespace := parts[0]
				podName := parts[1]
				podNamespaceMap[podName] = namespace
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("출력 스캔 오류: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return nil, nil, fmt.Errorf("명령어 종료 오류: %w", err)
	}

	return logURLs, podNamespaceMap, nil
}

func countInboundRequests(logURLs []string, podNamespaceMap map[string]string) map[string]struct {
	Namespace string
	Count     int
} {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	requestCounts := make(map[string]struct {
		Namespace string
		Count     int
	})
	for podName, namespace := range podNamespaceMap {
		requestCounts[podName] = struct {
			Namespace string
			Count     int
		}{Namespace: namespace, Count: 0}
	}

	fiveSecondsAgo := time.Now().Add(-5 * time.Second)

	for _, logURL := range logURLs {
		podFullName := strings.Split(strings.TrimSuffix(logURL, "/istio-proxy/0.log"), "/")[5]
		podNameParts := strings.Split(podFullName, "_")
		if len(podNameParts) < 2 {
			continue
		}

		podName := podNameParts[1]
		fmt.Println("로그 파일 가져오는 중:", logURL)

		// 새 로그 파일 시작점인지 확인
		resp, err := client.Get(logURL)
		if err != nil {
			fmt.Println("로그 파일 요청 오류:", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("로그 파일 응답 코드 오류: %d\n", resp.StatusCode)
			continue
		}

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("로그 파일 읽기 오류:", err)
			continue
		}

		logLines := strings.Split(string(body), "\n")

		// 마지막 읽은 위치부터 슬라이딩 윈도우
		offset := logOffsets[logURL]
		if offset >= int64(len(logLines)) {
			offset = 0 // 파일이 회전된 경우 초기화
		}

		// 새로운 로그 읽기
		for i := offset; i < int64(len(logLines)); i++ {
			line := logLines[i]
			if strings.Contains(line, "inbound") && (strings.Contains(line, "GET") || strings.Contains(line, "POST")) {
				timestampRegex := regexp.MustCompile(`\[(.*?)\]`)
				timestampMatch := timestampRegex.FindStringSubmatch(line)
				if len(timestampMatch) < 2 {
					continue
				}

				timestampStr := timestampMatch[1]
				timestamp, err := time.Parse("2006-01-02T15:04:05.000Z", timestampStr)
				if err == nil && timestamp.After(fiveSecondsAgo) {
					countData := requestCounts[podName]
					countData.Count++
					requestCounts[podName] = countData
				}
			}
		}
		logOffsets[logURL] = int64(len(logLines))
	}

	return requestCounts
}

func main() {
	logURLs, podNamespaceMap, err := getFilteredLogURLs()
	if err != nil {
		fmt.Println("Error fetching log URLs:", err)
		return
	}

	requestCounts := countInboundRequests(logURLs, podNamespaceMap)

	if len(requestCounts) == 0 {
		fmt.Println("최근 5초 동안 수신된 요청이 없습니다.")
		return
	}
	for pod, data := range requestCounts {
		fmt.Printf("%s (Namespace: %s): 최근 5초 동안 수신된 inbound 요청 수: %d\n", pod, data.Namespace, data.Count)
	}
}
