package logging

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// generateLogFile는 요청이 올 때마다 새로운 로그 파일을 생성하고 로그를 기록.
func GenerateLogFile() *os.File {
	// 한국 시간대 설정
	loc, _ := time.LoadLocation("Asia/Seoul")
	now := time.Now().In(loc)

	// 파일 이름을 현재 시간으로 설정
	filename := filepath.Join("etc/logDatas", now.Format("2006-01-02_15-04-05")+".log")

	// 파일 생성
	logFile, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to create log file: %v", err)
	}

	// 첫 줄에 현재 시간 기록
	logFile.WriteString("Log start time: " + now.Format("2006-01-02 15:04:05") + "\n")
	logFile.WriteString("---------------------------------------------\n")
	logFile.WriteString(fmt.Sprintf("\n\n"))
	log.Print("Log start time: " + now.Format("2006-01-02 15:04:05"))
	log.Print("---------------------------------------------")
	return logFile
}

// logAndWrite 함수는 주어진 문자열을 logFile에 기록하고, 동시에 로그로 출력
func LogMessage(logFile *os.File, message string) {
	logFile.WriteString(message)
	log.Print(message)
}

func Alert(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(message)))
}
