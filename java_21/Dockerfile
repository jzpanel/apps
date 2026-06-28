FROM eclipse-temurin:21-jdk

# 共享运行时容器：容器内用 supervisor 管理多个 Java 网站进程（每个 JVM 一个进程）。
# eclipse-temurin 基于 Ubuntu，用 apt 安装 supervisor。
# 注意：每个 Java 网站是独立 JVM，内存占用较大，共享容器需注意资源限制。
RUN printf 'Acquire::Retries "8";\n' > /etc/apt/apt.conf.d/80-retries \
    && apt-get update \
    && apt-get install -y --no-install-recommends supervisor curl tini \
    && mkdir -p /etc/supervisor/conf.d /var/log/supervisor \
    && rm -rf /var/lib/apt/lists/*

COPY init/supervisord.conf /etc/supervisor/supervisord.conf

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["supervisord", "-n", "-c", "/etc/supervisor/supervisord.conf"]
