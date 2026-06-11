# PHP 5.4 官方镜像 php:5.4-fpm 是已废弃的 Docker schema 1 manifest，BuildKit 拒绝以它为 FROM。
# CI 在构建前用 skopeo 把官方镜像无损转换为 schema 2，推到 jzpanel/php:5.4-fpm-base，
# 这里 FROM 转换后的基底（见 .github/workflows/build-php.yml 的 skopeo 步骤）。
FROM jzpanel/php:5.4-fpm-base

# php:5.4-fpm 基于 Debian Jessie，Jessie 已 EOL 且 apt 源已归档到 archive.debian.org。
# 必须把 sources.list 指向归档源，并关闭有效期校验（归档源 Release 文件已过期），否则 apt-get 全部 404。
RUN set -eux; \
    printf 'deb http://archive.debian.org/debian jessie main\ndeb http://archive.debian.org/debian-security jessie/updates main\n' > /etc/apt/sources.list; \
    printf 'Acquire::Check-Valid-Until "false";\n' > /etc/apt/apt.conf.d/99no-check-valid; \
    apt-get update

# 编译工具链 + 系统依赖（Debian Jessie 包名）
# docker-php-ext-install 编译扩展需要 gcc/make/autoconf，显式安装 build-essential 与 autoconf 保证可用。
# 注意：PHP 5.4+ 的 mbstring 自带 bundled oniguruma，无需外部 libonig
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

# gd：PHP 5.4 使用旧版 configure 参数
RUN docker-php-ext-configure gd \
        --with-gd \
        --with-freetype-dir=/usr/include/ \
        --with-jpeg-dir=/usr/include/ \
        --with-png-dir=/usr/include/

# 核心扩展逐个安装（PHP 5.4 镜像自带的 docker-php-ext-install 脚本太老，
# 不支持 -j 并行参数，传 -j 会被当作未知参数打印 usage 并退出 1，因此这里不能加 -j）
RUN docker-php-ext-install mysqli
RUN docker-php-ext-install pdo_mysql
RUN docker-php-ext-install gd
RUN docker-php-ext-install mbstring
RUN docker-php-ext-install exif
RUN docker-php-ext-install bcmath
RUN docker-php-ext-install sockets
RUN docker-php-ext-install soap
# 注意：intl 在 PHP 5.4 + Debian Jessie 的 ICU 版本组合下编译不稳定，
# 不在镜像内置；用户如需 intl 可通过面板"扩展管理"按需安装。

# PECL 扩展：redis（PHP 5.4 锁定 redis 2.2.8，4.x/5.x 在 5.4 上编译失败）
RUN pecl install redis-2.2.8 \
    && docker-php-ext-enable redis \
    && rm -rf /tmp/pear

# 说明：PHP 5.4 不内置 opcache，且 PECL 上的旧版 zendopcache（1.x）已下架无法下载，
# 新版 zendopcache(7.x) 不兼容 5.4。PHP 5.4 仅作老站点兼容用途，不提供 opcache 字节码缓存。
# 用户如确需，可自行在容器内编译，但不建议（5.4 已 EOL 多年）。

# 注意：实际监听端口由宿主机挂载的 php-fpm.conf 中 listen 字段决定（php_54=9054）
CMD ["php-fpm"]
