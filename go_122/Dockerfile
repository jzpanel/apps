FROM golang:1.22-alpine

# 共享运行时容器：容器内用 supervisor 管理多个 Go 网站进程。
# 容器内含完整 Go toolchain，支持两种启动方式：
#   1. go run .              （开发，启动稍慢）
#   2. ./<编译好的二进制>     （生产，推荐）
RUN apk add --no-cache supervisor bash curl tini git \
    && mkdir -p /etc/supervisor/conf.d /var/log/supervisor

# 注意：不在镜像里烤死 Go module 国内代理（GOPROXY）。
# 是否走国内代理由面板按「网络环境探测 + 用户设置」动态决定：
# 大陆/受限网络下面板会用 docker exec -e GOPROXY=https://goproxy.cn,direct 注入（依赖安装），
# 并在运行进程环境里注入（go run 运行时拉取 module）；国际网络通畅（境外服务器）走官方
# proxy.golang.org，避免被强制走国内源反而更慢。用户也可在网站「环境变量」里自设 GOPROXY 覆盖。

COPY init/supervisord.conf /etc/supervisor/supervisord.conf

RUN rm -rf /var/cache/apk/*

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["supervisord", "-n", "-c", "/etc/supervisor/supervisord.conf"]
