FROM node:20-alpine

# 共享运行时容器：容器内用 supervisor 管理多个网站的 Node 进程。
# 面板通过挂载的 /etc/supervisor/conf.d 写入每个网站的 program 配置，
# 再 docker exec supervisorctl reread/update 启动对应网站进程。
RUN apk add --no-cache supervisor bash curl tini \
    && mkdir -p /etc/supervisor/conf.d /var/log/supervisor

# 常用全局包（可选，方便用户）：pm2 不再需要（用 supervisor），保留 npm/yarn/pnpm
RUN corepack enable || true

# supervisord 主配置（include conf.d，由宿主机挂载覆盖 conf.d）
COPY init/supervisord.conf /etc/supervisor/supervisord.conf

RUN rm -rf /var/cache/apk/*

# tini 作为 PID 1，正确转发信号、回收僵尸进程
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["supervisord", "-n", "-c", "/etc/supervisor/supervisord.conf"]
