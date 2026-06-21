FROM php:5.6-fpm-alpine

# 系统依赖（编译各扩展所需的库）
RUN apk add --no-cache \
        libpng-dev \
        libjpeg-turbo-dev \
        libwebp-dev \
        freetype-dev \
        libzip-dev \
        icu-dev \
        libxml2-dev \
        linux-headers \
        bash \
        curl

# gd：PHP 5.x 使用旧版 configure 参数（--with-*-dir）
RUN docker-php-ext-configure gd \
        --with-gd \
        --with-freetype-dir=/usr/include/ \
        --with-jpeg-dir=/usr/include/ \
        --with-png-dir=/usr/include/ \
        --with-webp-dir=/usr/include/

# 内置扩展（PHP 5.6 内置 opcache，无 sodium/intl 编译问题）
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
        mbstring \
        opcache

# PECL 扩展：redis（PHP 5.x 必须锁定 redis 4.3.0，更高版本要求 PHP>=7.0）
RUN apk add --no-cache --virtual .pecl-deps $PHPIZE_DEPS \
    && pecl install redis-4.3.0 \
    && docker-php-ext-enable redis \
    && apk del .pecl-deps \
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

RUN rm -rf /var/cache/apk/*

# 注意：实际监听端口由宿主机挂载的 php-fpm.conf 中 listen 字段决定（php_56=9056）
# 移除官方镜像 zz-docker.conf 的 listen=9000（它按字母序在 www.conf 之后加载，
# 会覆盖面板挂载的 php-fpm.conf 的 listen，导致所有 PHP 版本都绑 9000 互相冲突）。
# 仅保留 daemonize=no（容器前台运行必需），监听端口完全由挂载的 php-fpm.conf 决定。
RUN printf '[global]\ndaemonize = no\n' > /usr/local/etc/php-fpm.d/zz-docker.conf

CMD ["php-fpm"]
