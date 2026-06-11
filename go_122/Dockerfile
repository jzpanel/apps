FROM golang:1.22-alpine

# 共享运行时容器：容器内用 supervisor 管理多个 Go 网站进程。
# 容器内含完整 Go toolchain，支持两种启动方式：
#   1. go run .              （开发，启动稍慢）
#   2. ./<编译好的二进制>     （生产，推荐）
RUN apk add --no-cache supervisor bash curl tini git \
    && mkdir -p /etc/supervisor/conf.d /var/log/supervisor

# 国内 Go module 代理（用户可在网站环境变量覆盖 GOPROXY）
ENV GOPROXY=https://goproxy.cn,direct

COPY init/supervisord.conf /etc/supervisor/supervisord.conf

RUN rm -rf /var/cache/apk/*

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["supervisord", "-n", "-c", "/etc/supervisor/supervisord.conf"]
