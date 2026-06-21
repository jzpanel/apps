FROM php:7.4-fpm-alpine

# 系统依赖（编译各扩展所需的库）
RUN apk add --no-cache \
        libpng-dev \
        libjpeg-turbo-dev \
        libwebp-dev \
        freetype-dev \
        libzip-dev \
        icu-dev \
        libxml2-dev \
        oniguruma-dev \
        linux-headers \
        bash \
        curl

# gd 需要在 install 前 configure
RUN docker-php-ext-configure gd \
        --with-freetype \
        --with-jpeg \
        --with-webp

# 内置扩展（一次 RUN 减少镜像层）
RUN docker-php-ext-install -j$(nproc) \
        mysqli \
        pdo_mysql \
        gd \
        zip \
        intl \
        soap \
        sockets \
        exif \
        bcmath \
        pcntl \
        opcache

# PECL 扩展：redis
RUN apk add --no-cache --virtual .pecl-deps $PHPIZE_DEPS \
    && pecl install redis \
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

# 注意：实际监听端口由宿主机挂载的 php-fpm.conf 中 listen 字段决定（php_74=9074）
# 移除官方镜像 zz-docker.conf 的 listen=9000（它按字母序在 www.conf 之后加载，
# 会覆盖面板挂载的 php-fpm.conf 的 listen，导致所有 PHP 版本都绑 9000 互相冲突）。
# 仅保留 daemonize=no（容器前台运行必需），监听端口完全由挂载的 php-fpm.conf 决定。
RUN printf '[global]\ndaemonize = no\n' > /usr/local/etc/php-fpm.d/zz-docker.conf

CMD ["php-fpm"]
