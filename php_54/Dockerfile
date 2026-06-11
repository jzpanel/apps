FROM php:5.4-fpm

# PHP 5.4 官方镜像基于 Debian Jessie，Jessie 已 EOL 且 apt 源已归档到 archive.debian.org。
# 必须把 sources.list 指向归档源，并关闭有效期校验（归档源的 Release 文件已过期），否则 apt-get 全部 404。
RUN set -eux; \
    printf 'deb http://archive.debian.org/debian jessie main\ndeb http://archive.debian.org/debian-security jessie/updates main\n' > /etc/apt/sources.list; \
    printf 'Acquire::Check-Valid-Until "false";\n' > /etc/apt/apt.conf.d/99no-check-valid; \
    apt-get update

# 系统依赖（Debian 包名，与 Alpine 不同）
RUN apt-get install -y --no-install-recommends --force-yes \
        libpng12-dev \
        libjpeg-dev \
        libwebp-dev \
        libfreetype6-dev \
        libicu-dev \
        libxml2-dev \
        libcurl4-openssl-dev \
    && rm -rf /var/lib/apt/lists/*

# gd：PHP 5.4 使用旧版 configure 参数
RUN docker-php-ext-configure gd \
        --with-gd \
        --with-freetype-dir=/usr/include/ \
        --with-jpeg-dir=/usr/include/ \
        --with-png-dir=/usr/include/

# 内置扩展（PHP 5.4 无 opcache 内置扩展 → 用 PECL 的 zendopcache；mbstring/intl 等可编译）
RUN docker-php-ext-install -j$(nproc) \
        mysqli \
        pdo_mysql \
        gd \
        intl \
        soap \
        sockets \
        exif \
        bcmath \
        pcntl \
        mbstring

# PECL 扩展：redis（PHP 5.4 锁定 redis 2.2.8，4.x 在 5.4 上编译易失败）
RUN pecl install redis-2.2.8 \
    && docker-php-ext-enable redis \
    && rm -rf /tmp/pear

# opcache：PHP 5.4 无内置 opcache，使用 PECL zendopcache 1.3.2
RUN pecl install zendopcache-1.3.2 \
    && echo 'zend_extension=opcache.so' > /usr/local/etc/php/conf.d/00-opcache.ini \
    && { \
        echo 'opcache.enable=1'; \
        echo 'opcache.memory_consumption=128'; \
        echo 'opcache.interned_strings_buffer=8'; \
        echo 'opcache.max_accelerated_files=10000'; \
        echo 'opcache.revalidate_freq=60'; \
        echo 'opcache.fast_shutdown=1'; \
    } >> /usr/local/etc/php/conf.d/00-opcache.ini \
    && rm -rf /tmp/pear

# 注意：实际监听端口由宿主机挂载的 php-fpm.conf 中 listen 字段决定（php_54=9054）
CMD ["php-fpm"]
