FROM golang:1.25-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /build/opstaskmanager ./cmd/server/

FROM alpine:3.21

RUN apk upgrade --no-cache && apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /build/opstaskmanager .

RUN mkdir -p /app/data

EXPOSE 5003

# 通过 TASK_MANAGER_* 环境变量或 .env 文件注入配置（见 README）
ENTRYPOINT ["./opstaskmanager"]
