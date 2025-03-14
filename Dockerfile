# 使用官方 Golang 镜像作为构建环境
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 复制 go mod 和 sum 文件
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o hraftd .

# 使用轻量级镜像
FROM alpine:latest

# 安装必要的工具
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# 从构建阶段复制可执行文件
COPY --from=builder /app/hraftd .

# 暴露必要的端口
EXPOSE 11000 12000 2379

# 设置容器启动命令
ENTRYPOINT ["./hraftd"]

# 默认参数
CMD ["-h"]
