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

# 国内 pip 源（用户可在网站环境变量覆盖）
ENV PIP_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple

COPY init/supervisord.conf /etc/supervisor/supervisord.conf

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["supervisord", "-n", "-c", "/etc/supervisor/supervisord.conf"]
