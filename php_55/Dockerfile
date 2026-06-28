# PHP 5.5 的官方 alpine 镜像（php:5.5-fpm-alpine）是单 manifest（非 manifest list），
# buildx 解析其 digest 时报 "content digest : not found"（buildkit 已知问题，no-cache/skopeo 复制均无效）。
# 因此 5.5 改用 Debian 版基底 php:5.5-fpm。该镜像是 schema 1，BuildKit 拒绝直接 FROM，
# 故 CI 先用 skopeo 无损转换为 schema 2 推到 jzpanel/php:5.5-fpm-base，这里 FROM 转换后的基底
# （见 .github/workflows/build-php.yml 的 skopeo 步骤）。
FROM jzpanel/php:5.5-fpm-base

# php:5.5-fpm 基于 Debian Jessie，Jessie 已 EOL 且 apt 源已归档到 archive.debian.org。
RUN set -eux; \
    printf 'deb http://archive.debian.org/debian jessie main\ndeb http://archive.debian.org/debian-security jessie/updates main\n' > /etc/apt/sources.list; \
    printf 'Acquire::Check-Valid-Until "false";\nAcquire::Retries "8";\n' > /etc/apt/apt.conf.d/99no-check-valid; \
    apt-get update

# 编译工具链 + 系统依赖（Debian Jessie 包名）
RUN apt-get install -y --no-install-recommends --force-yes \
        build-essential \
        autoconf \
        pkg-config \
        libpng12-dev \
        libjpeg-dev \
        libwebp-dev \
        libfreetype6-dev \
        libxml2-dev \
        libcurl4-openssl-dev \
    && rm -rf /var/lib/apt/lists/*

# gd：PHP 5.5 使用旧版 configure 参数
RUN docker-php-ext-configure gd \
        --with-gd \
        --with-freetype-dir=/usr/include/ \
        --with-jpeg-dir=/usr/include/ \
        --with-png-dir=/usr/include/

# 核心扩展逐个安装（PHP 5.5 镜像自带的 docker-php-ext-install 脚本不支持 -j 参数）
RUN docker-php-ext-install mysqli
RUN docker-php-ext-install pdo_mysql
RUN docker-php-ext-install gd
RUN docker-php-ext-install mbstring
RUN docker-php-ext-install exif
RUN docker-php-ext-install bcmath
RUN docker-php-ext-install sockets
RUN docker-php-ext-install soap
# opcache：PHP 5.5 起 opcache 进入核心，可直接编译
RUN docker-php-ext-install opcache

# PECL 扩展：redis（PHP 5.5 锁定 redis 2.2.8）
RUN pecl install redis-2.2.8 \
    && docker-php-ext-enable redis \
    && rm -rf /tmp/pear

# opcache 推荐配置
RUN { \
        echo 'opcache.enable=1'; \
        echo 'opcache.memory_consumption=128'; \
        echo 'opcache.interned_strings_buffer=8'; \
        echo 'opcache.max_accelerated_files=10000'; \
        echo 'opcache.revalidate_freq=60'; \
        echo 'opcache.fast_shutdown=1'; \
    } > /usr/local/etc/php/conf.d/opcache-recommended.ini

# 注意：实际监听端口由宿主机挂载的 php-fpm.conf 中 listen 字段决定（php_55=9055）
# 移除官方镜像 zz-docker.conf 的 listen=9000（它按字母序在 www.conf 之后加载，
# 会覆盖面板挂载的 php-fpm.conf 的 listen，导致所有 PHP 版本都绑 9000 互相冲突）。
# 仅保留 daemonize=no（容器前台运行必需），监听端口完全由挂载的 php-fpm.conf 决定。
RUN mkdir -p /usr/local/etc/php-fpm.d && printf '[global]\ndaemonize = no\n' > /usr/local/etc/php-fpm.d/zz-docker.conf

CMD ["php-fpm"]
