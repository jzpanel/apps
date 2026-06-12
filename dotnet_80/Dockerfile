FROM mcr.microsoft.com/dotnet/sdk:8.0

# 共享运行时容器：容器内用 supervisor 管理多个 .NET 网站进程。
# 用 SDK 镜像（含完整工具链），支持 dotnet build/publish 和直接 dotnet *.dll 运行。
# 基于 Debian，用 apt 安装 supervisor。
RUN apt-get update \
    && apt-get install -y --no-install-recommends supervisor curl tini \
    && mkdir -p /etc/supervisor/conf.d /var/log/supervisor \
    && rm -rf /var/lib/apt/lists/*

# 关闭首次运行遥测与欢迎信息
ENV DOTNET_CLI_TELEMETRY_OPTOUT=1
ENV DOTNET_SKIP_FIRST_TIME_EXPERIENCE=1

COPY init/supervisord.conf /etc/supervisor/supervisord.conf

ENTRYPOINT ["/usr/bin/tini", "--"]
CMD ["supervisord", "-n", "-c", "/etc/supervisor/supervisord.conf"]
