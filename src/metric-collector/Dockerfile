# 1. 빌드 단계
FROM golang:1.22.6 AS builder

# 작업 디렉터리 설정
WORKDIR /app

# 모듈 파일 복사 (있는 경우)
COPY go.mod go.sum ./

# 의존성 다운로드
RUN go mod download

# 소스 코드 복사
COPY . .

# 애플리케이션 빌드
RUN CGO_ENABLED=0 GOOS=linux go build -o app .

# 2. 실행 단계
FROM alpine:latest

# 작업 디렉터리 설정
WORKDIR /root/

# 빌드된 바이너리 복사
COPY --from=builder /app/app .

# 애플리케이션 실행 명령어
CMD ["./app"]
