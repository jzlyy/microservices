FROM golang:alpine AS builder

WORKDIR /app

# 复制依赖文件并下载
COPY go.mod go.sum ./
RUN go mod download

# 复制源码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -o order-service

# 最终阶段
FROM alpine:3.21.3

# 安装SSL证书
RUN apk --no-cache add ca-certificates

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /app/order-service .

# 暴露端口
EXPOSE 8080

# 设置环境变量 (可在运行时覆盖)
ENV DB_HOST=host.docker.internal \
    DB_PORT=3306 \
    DB_USER=root \
    DB_NAME=ecommerce

# 启动应用
CMD ["./order-service"]