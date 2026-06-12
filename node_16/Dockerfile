FROM node:16-alpine

# 共享运行时容器：容器内用 supervisor 管理多个网站的 Node 进程。
# 面板通过挂载的 /etc/supervisor/conf.d 写入每个网站的 program 配置，
# 再 docker exec supervisorctl reread/update 启动对应网站进程。
RUN apk add --no-cache supervisor bash curl tini \
    && mkdir -p /etc/supervisor/conf.d /var/log/supervisor

RUN corepack enable || true

COPY init/supervisord.conf /etc/supervisor/supervisord.conf

RUN rm -rf /var/cache/apk/*

ENTRYPOINT ["/sbin/tini", "--"]
CMD ["supervisord", "-n", "-c", "/etc/supervisor/supervisord.conf"]
