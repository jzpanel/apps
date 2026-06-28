FROM python:3.11-slim

# 共享运行时容器：容器内用 supervisor 管理多个 Python 网站进程。
# 面板通过挂载的 /etc/supervisor/conf.d 写入每个网站的 program 配置，
# 再 docker exec supervisorctl reread/update 启动对应网站进程。
# 用 slim（Debian）而非 alpine：Python C 扩展（如 cryptography、psycopg2）在 alpine 上常需源码编译，slim 兼容性更好。
RUN apt-get update \
    && apt-get install -y --no-install-recommends supervisor curl tini \
    && mkdir -p /etc/supervisor/conf.d /var/log/supervisor \
    && pip install --no-cache-dir gunicorn uvicorn \
    && rm -rf /var/lib/apt/lists/*

# 注意：不在镜像里烤死 pip 国内源（PIP_INDEX_URL）。
# 是否走国内镜像由面板按「网络环境探测 + 用户设置」动态决定：
# 大陆/受限网络安装依赖时面板会用 docker exec -e PIP_INDEX_URL=... 注入国内源；
# 国际网络通畅（境外服务器）则走官方 pypi.org，避免被强制走国内源反而更慢。
# 用户也可在网站「环境变量」里自行设置 PIP_INDEX_URL 覆盖。

COPY init/supervisord.conf /etc/supervisor/supervisord.conf

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["supervisord", "-n", "-c", "/etc/supervisor/supervisord.conf"]
