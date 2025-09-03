# syntax=docker/dockerfile:1.7

# ---------- Build stage ----------
# 使用官方的 Go 镜像作为构建环境
ARG GO_VERSION=1.24
FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /src

# 优先缓存依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建静态二进制文件，以实现最小的运行时镜像
ARG TARGETOS
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w" -o /out/gcli2api .

# ---------- Runtime stage ----------
# 使用轻量级的 Alpine 镜像作为最终的运行环境
FROM alpine:3.20 AS runtime

WORKDIR /app

# 默认启动命令，指向 Render 环境中将要创建的配置文件
ENTRYPOINT ["/app/gcli2api"]
CMD ["server", "-c", "/app/config.json"]

# 暴露服务端口
EXPOSE 8085

# 安装 ca-certificates 以支持 HTTPS 通信
RUN apk --no-cache add ca-certificates

# 【关键修改】移除：删除非 root 用户创建和切换的步骤
# 容器将以默认的 root 用户运行，以避免在 Render 环境中出现权限问题
# RUN adduser -D -H -u 10001 appuser
# USER 10001

# 从构建阶段复制编译好的二进制文件
COPY --from=builder /out/gcli2api /app/gcli2api
# 复制启动脚本并赋予执行权限
COPY render-start.sh /app/render-start.sh
RUN chmod +x /app/render-start.sh
