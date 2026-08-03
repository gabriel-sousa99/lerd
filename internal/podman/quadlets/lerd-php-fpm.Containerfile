FROM docker.io/library/composer:latest AS composer-bin

# Builder stage: compile every PHP extension so the build toolchain
# never lands in the final image. .so files and configs travel into
# the runtime stage via COPY at the bottom.
FROM docker.io/library/php:{{.Version}}-fpm-alpine AS builder

RUN apk update && apk add --no-cache \
        autoconf \
        make \
        g++ \
        git \
        linux-headers \
        curl-dev \
        libzip-dev \
        libpng-dev \
        libjpeg-turbo-dev \
        freetype-dev \
        libwebp-dev \
        icu-dev \
        oniguruma-dev \
        libxml2-dev \
        postgresql-dev \
        imagemagick-dev \
        gmp-dev \
        bzip2-dev \
        openldap-dev \
        sqlite-dev \
        libxslt-dev \
        zlib-dev \
    && PHP_ID="$(php -r 'echo PHP_VERSION_ID;')" \
    && if [ "$PHP_ID" -lt 70400 ]; then \
           docker-php-ext-configure gd --with-freetype-dir=/usr --with-jpeg-dir=/usr --with-png-dir=/usr --with-webp-dir=/usr; \
       else \
           docker-php-ext-configure gd --with-freetype --with-jpeg --with-webp; \
       fi \
    && docker-php-ext-install -j$(nproc) \
        curl \
        pdo_mysql \
        pdo_pgsql \
        bcmath \
        mbstring \
        xml \
        zip \
        gd \
        intl \
        pcntl \
        exif \
        ftp \
        sockets \
        gmp \
        bz2 \
        calendar \
        dba \
        ldap \
        mysqli \
        soap \
        shmop \
        sysvmsg \
        sysvsem \
        sysvshm \
        xsl \
    && (docker-php-ext-enable opcache || true) \
    && (pecl channel-update pecl.php.net || true) \
    && if [ "$PHP_ID" -lt 70000 ]; then REDIS_PKG=redis-4.3.0; \
         elif [ "$PHP_ID" -lt 70400 ]; then REDIS_PKG=redis-5.3.7; \
         else REDIS_PKG=redis; fi \
    && { (yes '' | pecl install "$REDIS_PKG" && docker-php-ext-enable redis) \
         || (git clone --depth 1 https://github.com/phpredis/phpredis /tmp/phpredis \
             && cd /tmp/phpredis && phpize && ./configure && make -j$(nproc) && make install \
             && docker-php-ext-enable redis \
             && rm -rf /tmp/phpredis) \
         || true; } \
    && { (yes '' | pecl install imagick && docker-php-ext-enable imagick) \
         || (git clone --depth 1 https://github.com/Imagick/imagick /tmp/imagick \
             && cd /tmp/imagick && phpize && ./configure && make -j$(nproc) && make install \
             && docker-php-ext-enable imagick \
             && rm -rf /tmp/imagick) \
         || true; } \
    && { (yes '' | pecl install igbinary && docker-php-ext-enable igbinary) || true; } \
    && { (PHPVER="$(php -r 'echo PHP_MAJOR_VERSION,".",PHP_MINOR_VERSION;')" \
          && if [ "$PHPVER" = "7.4" ]; then yes '' | pecl install mongodb-1.16.2; \
             else yes '' | pecl install mongodb; fi \
          && docker-php-ext-enable mongodb) || true; } \
    && { (yes '' | pecl install pcov && docker-php-ext-enable pcov) || true; } \
    && { (apk add --no-cache libmemcached-dev zlib-dev \
          && yes '' | pecl install memcached && docker-php-ext-enable memcached) || true; } \
    && { (apk add --no-cache rabbitmq-c-dev \
          && yes '' | pecl install amqp && docker-php-ext-enable amqp) || true; } \
    && { (git clone --depth 1 --branch release/latest https://github.com/NoiseByNorthwest/php-spx /tmp/php-spx \
          && cd /tmp/php-spx && phpize && ./configure && make -j$(nproc) && make install \
          && docker-php-ext-enable spx) || true; } \
    && mkdir -p /usr/local/share/misc/php-spx/assets/web-ui \
    && rm -rf /tmp/php-spx /tmp/pear /var/cache/apk/*

# Xdebug compiled in the builder too. Legacy PHP needs older xdebug majors.
# 5.6 stops at xdebug 2.5.5 (last release with 5.x support).
RUN PHPVER="$(php -r 'echo PHP_MAJOR_VERSION,".",PHP_MINOR_VERSION;')" \
    && case "$PHPVER" in \
        5.6) XDEBUG_PKG="xdebug-2.5.5" ;; \
        7.2) XDEBUG_PKG="xdebug-3.1.6" ;; \
        7.4) XDEBUG_PKG="xdebug-3.1.6" ;; \
        8.0) XDEBUG_PKG="xdebug-3.3.2" ;; \
        *)   XDEBUG_PKG="xdebug" ;; \
    esac \
    && pecl channel-update pecl.php.net \
    && yes '' | pecl install "$XDEBUG_PKG" && docker-php-ext-enable xdebug \
    && rm -rf /tmp/pear /var/cache/apk/*

# ── Oracle Instant Client 21.18 + oci8 (builder) ────────────────────────────
# Lerd Oracle fork addition. Instant Client 21.18 is glibc-linked; Alpine is
# musl, so gcompat + libc6-compat provide the ABI shim. libaio/libnsl/
# libstdc++ are direct deps of libclntsh. The compiled oci8.so travels into
# the runtime stage via the existing COPY --from=builder block at the bottom;
# the Instant Client itself is copied separately in the runtime stage below.
# pecl package is pinned per-PHP-major where the rolling "oci8" tag drops
# support; PHP 8.2+ tracks the latest.
#
# 8.1 gets 3.2.1, not 3.3.0: 3.3.0's package.xml claims min PHP 8.1, but its
# generated oci8_arginfo.h references ZEND_STR_SENSITIVEPARAMETER and
# ZEND_ACC_ALLOW_DYNAMIC_PROPERTIES, both of which only exist from 8.2, so the
# build dies at compile time. The metadata is optimistic; the compiler is not.
#
# This image is x86_64-only by necessity: Oracle publishes no linux.arm64
# Instant Client for 21.x (only 19.x), and linking oci8 against the x64
# libclntsh on aarch64 fails with "skipping incompatible libclntsh.so".
# The arch gate comes first: the archives below are the x64 build, so on aarch64
# there is nothing worth fetching. The directory is still created because the
# runtime stage's COPY --from=builder cannot be made conditional; it just arrives
# empty, and no oci8 is enabled to look for it.
RUN set -eux; \
    ARCH="$(uname -m)"; \
    PHPVER="$(php -r 'echo PHP_MAJOR_VERSION,".",PHP_MINOR_VERSION;')"; \
    mkdir -p /opt/oracle/instantclient_21_18; \
    ln -sfn /opt/oracle/instantclient_21_18 /opt/oracle/instantclient; \
    if [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then \
      echo "Skipping Oracle Instant Client and OCI8 on ${ARCH}: Oracle publishes no linux.arm64 build for 21.x"; \
    else \
      apk add --no-cache libaio libnsl gcompat libc6-compat libstdc++ unzip; \
      cd /opt/oracle; \
      curl -fsSLO https://download.oracle.com/otn_software/linux/instantclient/2118000/instantclient-basic-linux.x64-21.18.0.0.0dbru.zip; \
      curl -fsSLO https://download.oracle.com/otn_software/linux/instantclient/2118000/instantclient-sdk-linux.x64-21.18.0.0.0dbru.zip; \
      unzip -qo instantclient-basic-linux.x64-21.18.0.0.0dbru.zip; \
      unzip -qo instantclient-sdk-linux.x64-21.18.0.0.0dbru.zip; \
      rm -f /opt/oracle/*.zip; \
      pecl channel-update pecl.php.net; \
      case "$PHPVER" in \
        5.6)             OCI8_PKG="oci8-2.0.12" ;; \
        7.2|7.3|7.4)     OCI8_PKG="oci8-2.2.0" ;; \
        8.0)             OCI8_PKG="oci8-3.0.1" ;; \
        8.1)             OCI8_PKG="oci8-3.2.1" ;; \
        8.2|8.3)         OCI8_PKG="oci8-3.3.0" ;; \
        8.4)             OCI8_PKG="oci8-3.4.1" ;; \
        *)               OCI8_PKG="oci8" ;; \
      esac; \
      printf "instantclient,/opt/oracle/instantclient\n" | pecl install "$OCI8_PKG"; \
      docker-php-ext-enable oci8; \
    fi; \
    rm -rf /opt/oracle/instantclient_21_18/sdk /tmp/pear /var/cache/apk/*

# lerd_devtools: lerd's engine-level Debug-window capture (queries, mail, views,
# events, jobs, http). Compiled in the builder so its .so and the
# docker-php-ext-enable conf.d travel into the runtime stage via the
# COPY --from=builder below, like every other extension, so users pull it
# ready-built instead of compiling C on their own machine. The marker line
# hashes the extension source so any change to it drifts the image hash and
# rebuilds the base; TestDevtoolsSourceMarkerInSync keeps the marker honest.
# No-op at runtime on PHP < 8.0 (no zend_observer); the || true degrades a
# compile failure to "Debug window unavailable" rather than bricking the image.
# lerd_devtools-src-sha256: 4d7c5e0c3032
COPY internal/podman/devtools /tmp/lerd-devtools
RUN { cd /tmp/lerd-devtools && phpize && ./configure --enable-lerd-devtools && make -j$(nproc) && make install && docker-php-ext-enable lerd_devtools; } || true; \
    rm -rf /tmp/lerd-devtools /var/cache/apk/*

# Project-defined custom extensions compile here while the toolchain
# is available. Their .so files travel through the COPY below.
{{.CustomExtensions}}

# ── Runtime stage ───────────────────────────────────────────────────────────
FROM docker.io/library/php:{{.Version}}-fpm-alpine

# Runtime libraries only (no -dev headers, no toolchain). PHP's
# compiled extensions dlopen these.
RUN apk update && apk add --no-cache \
        ghostscript \
        imagemagick \
        libgomp \
        ffmpeg \
        git \
        openssh-client \
        mysql-client \
        nodejs \
        npm \
        libzip \
        libpng \
        libjpeg-turbo \
        freetype \
        libwebp \
        icu-libs \
        oniguruma \
        libxml2 \
        libpq \
        gmp \
        bzip2 \
        libldap \
        sqlite-libs \
        libxslt \
        libmemcached-libs \
        rabbitmq-c \
        openssh-client \
    && rm -rf /var/cache/apk/*

# icu-data-full carries the full CLDR locale set for ext-intl (#332). Alpine
# 3.16+ ships it as a separate package; older bases fold the full data into
# icu-libs, so the package is absent there and the install is skipped.
RUN apk add --no-cache icu-data-full 2>/dev/null || true

# ── Oracle Instant Client 21.18 (runtime) ───────────────────────────────────
# Lerd Oracle fork addition. Runtime libs for oci8: glibc ABI shim (gcompat/
# libc6-compat) plus libclntsh's direct deps. The Instant Client itself is
# copied from the builder stage (sdk/ stripped there) and exposed via
# LD_LIBRARY_PATH so PHP can resolve libclntsh.so at extension load time.
RUN apk add --no-cache libaio libnsl gcompat libc6-compat libstdc++ \
    && rm -rf /var/cache/apk/*
# On Alpine 3.8 (PHP 5.6 base) musl doesn't expose libresolv.so.2 separately
# and Oracle libclntsh.so insists on dlopen'ing it. The shim is harmless on
# newer Alpine — gcompat already provides resolv.h symbols there, so this
# symlink is essentially a no-op except on the legacy tier. The musl soname
# carries the arch, and the target is checked first so a mismatch leaves no
# dangling link at a path the loader searches.
RUN MUSL="/lib/libc.musl-$(uname -m).so.1"; \
    if [ ! -e /lib/libresolv.so.2 ] && [ -e "$MUSL" ]; then ln -sf "$MUSL" /lib/libresolv.so.2; fi
COPY --from=builder /opt/oracle/instantclient_21_18 /opt/oracle/instantclient_21_18
RUN ln -sfn /opt/oracle/instantclient_21_18 /opt/oracle/instantclient
# TNS_ADMIN is the directory Instant Client reads tnsnames.ora, sqlnet.ora and an
# Autonomous wallet from, which is how an enterprise database is addressed: by
# alias rather than by host:port/service. The quadlet mounts the host's copy over
# this path read-only, so the directory exists here and is empty in the image.
#
# The symlink is for an unedited wallet: a downloaded Autonomous wallet's
# sqlnet.ora says DIRECTORY="?/network/admin", and Oracle expands "?" to
# $ORACLE_HOME, which here is the Instant Client dir. Pointing that path back at
# TNS_ADMIN means the wallet works as downloaded instead of requiring every user
# to find and rewrite that line.
RUN mkdir -p /opt/oracle/network/admin /opt/oracle/instantclient/network \
    && ln -sfn /opt/oracle/network/admin /opt/oracle/instantclient/network/admin
ENV ORACLE_HOME=/opt/oracle/instantclient \
    LD_LIBRARY_PATH=/opt/oracle/instantclient \
    TNS_ADMIN=/opt/oracle/network/admin

# Runtime system libs for user-configured custom extensions (e.g.
# imap needs c-client.so). Empty when no custom exts have apk deps.
{{.CustomExtensionsRuntime}}

# User-requested extra Alpine packages (lerd php:pkg). Empty until opted in.
{{.CustomPackages}}

# Compiled extensions + config from the builder stage; ~25 extensions
# plus xdebug + pecl modules without dragging autoconf/make/g++ across.
COPY --from=builder /usr/local/lib/php/extensions/ /usr/local/lib/php/extensions/
COPY --from=builder /usr/local/etc/php/conf.d/ /usr/local/etc/php/conf.d/

# SPX profiler web UI assets (shipped as files, not embedded in the .so). The
# builder's mkdir -p guarantees this path exists even if the SPX build failed.
COPY --from=builder /usr/local/share/misc/php-spx/ /usr/local/share/misc/php-spx/

# MariaDB client (mysql-client) connecting to lerd MySQL uses self-signed
# certs; disable SSL verification so CLI tools (mysqldump, schema loading)
# work out of the box.
RUN mkdir -p /etc/my.cnf.d && printf '[client]\nssl=0\n' > /etc/my.cnf.d/lerd-no-ssl.cnf

# Composer from the official image.
COPY --from=composer-bin /usr/bin/composer /usr/local/bin/composer

# xdebugctl — Xdebug's control-socket CLI (Xdebug >= 3.3). `lerd xdebug pause`
# uses it to break the IDE debugger into a running worker/CLI process on demand.
# Pinned by SHA-256; bump BOTH hashes when upstream updates the binary. A bad
# checksum fails the build (the binary changed under us); a network/arch miss is
# tolerated and just leaves the pause feature unavailable in that image.
RUN set -eu; \
    apk add --no-cache --virtual .xdbgctl-dl ca-certificates wget; \
    case "$(uname -m)" in \
      x86_64)  U="https://xdebug.org/files/binaries/xdebugctl";       S="dbfe72bdb4e23e2245305b14cce2931cc86db40061b830d15f801d1249c4d3c8" ;; \
      aarch64) U="https://xdebug.org/files/binaries/xdebugctl-arm64"; S="48543fb8aaae273c161efa05e07259cd49072a3ba0ad06669baa71b3f7f3ff32" ;; \
      *)       U="" ;; \
    esac; \
    if [ -n "$U" ]; then \
      if wget -qO /tmp/xdebugctl "$U"; then \
        echo "$S  /tmp/xdebugctl" | sha256sum -c -; \
        install -m 0755 /tmp/xdebugctl /usr/local/bin/xdebugctl; \
        rm -f /tmp/xdebugctl; \
      else \
        echo "WARN: xdebugctl download failed; 'lerd xdebug pause' disabled in this image"; \
      fi; \
    fi; \
    apk del .xdbgctl-dl

# Interactive shell for lerd shell. zsh/fzf exist on every alpine base;
# bat lands on 3.16+ and starship/eza/zoxide on 3.18+, so the optional
# tools install tolerantly and zshrc inits starship only when present.
RUN apk add --no-cache zsh fzf \
    && { apk add --no-cache bat 2>/dev/null || true; } \
    && { apk add --no-cache starship eza zoxide 2>/dev/null || true; } \
    && mkdir -p /etc/zsh /root/.zsh_state \
    && printf 'export EDITOR=vi\nexport PAGER=less\nexport HISTFILE=/root/.zsh_state/history\nexport HISTSIZE=10000\nexport SAVEHIST=10000\nsetopt INC_APPEND_HISTORY SHARE_HISTORY\nautoload -Uz compinit && compinit -u\nif command -v starship >/dev/null 2>&1; then\n  eval "$(starship init zsh)"\nfi\n' \
        > /etc/zsh/zshrc

# Override pool: run workers as root, log errors to stderr
RUN printf '[www]\nuser=root\ngroup=root\ncatch_workers_output=yes\nphp_flag[display_errors]=off\nphp_admin_value[error_log]=/proc/self/fd/2\nphp_admin_flag[log_errors]=on\n' > /usr/local/etc/php-fpm.d/zz-lerd.conf

{{.MkcertCA}}
