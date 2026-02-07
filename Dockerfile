# Build stage
FROM golang:1.25.4-alpine AS builder

WORKDIR /app

# Install dependencies
COPY go.mod go.sum ./
ENV GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
ENV GOSUMDB=off
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o IndicatorTask main/main.go

# Run stage
FROM alpine:3.19

RUN apk add --no-cache tzdata

WORKDIR /app

# Copy the binary from the build stage
COPY --from=builder /app/IndicatorTask .

# Config 目录由运行时挂载（见 README），此处仅创建空目录以便未挂载时路径存在
RUN mkdir -p config

CMD ["./IndicatorTask"]
