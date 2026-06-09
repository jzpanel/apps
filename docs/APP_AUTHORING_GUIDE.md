# 应用编写规范（App Authoring Guide）

本文档是为 jzpanel/apps 仓库添加新应用的**唯一权威指南**。新增应用前必须通读，已有应用修改时也建议查阅相应章节。

> 维护者：核心团队
> 适用范围：jzpanel/apps 仓库下所有应用
> 版本：v1.0（2026-05-25）
> 关联：spec/app-image-version-reliability

---

## 目录

1. [核心原则](#核心原则)
2. [文件结构](#文件结构)
3. [app.yaml 完整字段参考](#appyaml-完整字段参考)
4. [docker-compose.yml 模板规范](#docker-composeyml-模板规范)
5. [模板占位符（Placeholders）](#模板占位符placeholders)
6. [挂载目录权限（volumes_init）](#挂载目录权限volumes_init)
7. [初始化文件（init_files）](#初始化文件init_files)
8. [动态变量（compose_vars）](#动态变量compose_vars)
9. [服务注册与依赖](#服务注册与依赖)
10. [资源限制](#资源限制)
11. [反代网站（proxy_spec）](#反代网站proxy_spec)
12. [icon 与图标](#icon-与图标)
13. [CI 校验](#ci-校验)
14. [新增应用的标准流程](#新增应用的标准流程)
15. [常见错误与排查](#常见错误与排查)

---

## 核心原则

### 1. 一次声明，多处复用

所有应用相关的元数据都在 `app.yaml` 中**声明式**描述。后端读取 yaml 并驱动安装/迁移/管理 UI，前端通过 API 拿到 yaml 渲染界面。**不要**把任何 app 专用逻辑写到后端代码里。

### 2. 显式优于隐式

任何"魔法路径"、"魔法 UID"、"特殊处理"必须写成 yaml 字段。CI 可以校验，contributor 可以理解，AI 可以读懂。

### 3. 可重现可验证

每个版本的镜像 tag 必须是**精确版本**（如 `1.27.1.2-alpine`），不是 floating tag（如 `latest`、`1.27`）。CI 在 PR 时会验证 manifest 真实存在。

### 4. 国内可用

所有镜像必须能从 jzpanel 镜像加速地址（`docker.jzpanel.top` 等）拉到。如果上游只发布到 ghcr.io 等小众仓库，需要先沟通。

---

## 文件结构

```
apps-repo/
├── <app-key>/
│   ├── app.yaml                    # 主配置（必需）
│   ├── docker-compose.yml          # 容器编排模板（必需）
│   ├── icon.svg                    # 图标，svg 优先，128×128（必需）
│   ├── icon.png                    # 备用 png 图标（可选）
│   └── init/                       # 初始化文件目录（按需）
│       ├── postgresql.conf         # 配置文件
│       ├── pg_hba.conf
│       └── ...
```

**app-key 命名规则**：
- 全小写，下划线分隔（如 `postgresql`、`php_83`、`mongo_express`）
- 不允许连字符 `-`（compose 文件里 service name 也用 key，YAML 不喜欢 `-` 开头）
- 多版本共存的应用用版本后缀（PHP 7.4 → `php_74`）

---

## app.yaml 完整字段参考

下面是**全部支持的字段**，按使用频率排序。每个字段标注：
- `必需` / `推荐` / `可选`
- 字段含义、合法取值、举例

### 基础元数据

```yaml
key: postgresql                       # 必需 - 应用唯一标识，与目录名一致
name: PostgreSQL                      # 必需 - 展示名（前端列表显示）
category: database                    # 必需 - 分类，用于前端筛选
description: |                        # 必需 - 一行简短描述
  功能强大的开源对象关系型数据库
source: official                      # 推荐 - 应用来源标记
icon: icon.svg                        # 必需 - 图标文件名（与同级 icon.svg 对应）
```

#### `category` 合法值

| 值 | 含义 | 例 |
|----|------|-----|
| `runtime` | Web 服务器 / 运行时 | nginx, openresty, apache, php_* |
| `database` | 数据库 | mysql, postgresql, mongodb, redis |
| `tool` | 管理工具 | phpmyadmin, pgweb, mongoexpress |
| `service` | 后台服务 / 网络 | pureftpd, ssh-bastion |
| `cms` | 应用程序（一般用户面） | wordpress |

> **重要**：`category: database` 会激活两个特殊行为：
> - 安装时清空数据目录（确保 init password 生效）
> - 平滑迁移时强制要求用户勾选"已知晓数据兼容风险"

### 版本（versions）

```yaml
versions:
  - display: "16"                     # 必需 - 用户视角版本号
    tag: "16-alpine"                  # 必需 - 真实 Docker tag
    image: postgres                   # 必需 - 镜像主名（不含 tag）
    architectures: [amd64, arm64]     # 推荐 - 该版本支持的架构
    is_default: true                  # 可选 - 默认版本（最多一个 true）
    digest: "sha256:abcd..."          # 可选 - 强一致性校验
    deprecated: false                 # 可选 - 标记为已废弃版本
    rolling: false                    # 可选 - rolling tag（如 latest）
    released: "2024-09-26"            # 可选 - 发布日期，仅展示
default_version: "16"                 # 必需 - 默认版本号（display 字段）
architectures: [amd64, arm64]         # 推荐 - 应用级 fallback 架构
```

#### 版本字段详解

| 字段 | 必需 | 说明 |
|------|------|------|
| `display` | ✓ | 用户在前端下拉框看到的字符串。多版本时要简洁（"16"、"16.4"），别把 `-alpine` 后缀也带上 |
| `tag` | ✓ | 实际 docker pull 用的 tag，**必须是精确不变 tag**（不要 `latest`、不要 `16`） |
| `image` | ✓ | 镜像主名。Renovate 会用这个字段查 Docker Hub 自动 PR 升级 |
| `architectures` | ✓ | 该版本支持的 CPU 架构。`amd64+arm64` 默认，少数老镜像（如 mysql 5.7）只有 amd64 |
| `is_default` | optional | 标记默认安装版本。同一个应用最多一个 true。未声明时取 `default_version` 字段 |
| `deprecated` | optional | 老版本（已 EOL）置 true。前端下拉里会有警告，CI `--strict` 模式会校验 |
| `rolling` | optional | 滚动 tag（latest、stable）置 true。CI 跳过 manifest 校验，安装时警告用户 |
| `released` | optional | 仅展示用，对功能无影响 |
| `digest` | optional | sha256 强一致性校验。CI 校验时若声明了 digest 必须匹配真实 digest |

**精确版本 tag 选择规则：**

```yaml
# ✓ 推荐：精确到 patch
tag: "16.4-alpine"       # postgres
tag: "8.0.39"            # mysql
tag: "1.27.1.2-alpine"   # openresty

# 不推荐：浮动 tag（CI 校验会过，但每次拉取内容可能变）
tag: "16-alpine"
tag: "8.0"
```

**多版本如何决定？**

- 至少包含 `current LTS` + `current stable`（2~4 个常见版本）
- 不要超过 6 个版本（前端下拉拥挤，维护成本高）
- EOL 版本标 `deprecated: true`，半年后从 yaml 删除

---

### service_type 与 db_type

```yaml
service_type: database               # 服务类型，影响卸载检查、reload 行为
db_type: pgsql                       # service_type=database 时必须
```

#### `service_type` 合法值

| 值 | 含义 | 行为差异 |
|----|------|---------|
| `webserver` | Web 服务器 | 卸载时检查是否有网站使用此 ws；reload 用 `kill -HUP 1` |
| `database` | 数据库 | 安装前清数据目录；卸载时检查是否有数据库使用此服务 |
| `php` | PHP 运行时 | 卸载时检查是否有网站使用此 PHP；reload 用 `kill -HUP 1` |
| `tool` | 管理工具 | 无特殊处理 |
| `service` | 后台服务 | 无特殊处理 |
|（留空）| 默认 | 无特殊处理 |

#### `db_type` 合法值

仅 `service_type: database` 时使用。前端创建数据库时按此筛选可用服务。

| 值 | 应用 |
|----|------|
| `mysql` | mysql, mariadb |
| `pgsql` | postgresql |
| `mongodb` | mongodb |
| `redis` | redis |

### 健康检查超时

```yaml
health_check_timeout: 60             # 秒，安装时等待容器 running 的最大时间
```

| 应用类型 | 推荐值 |
|---------|--------|
| Web 服务器 | 30 秒 |
| 数据库（首次 init） | 60–120 秒（init 数据目录较慢） |
| 重型应用（elasticsearch 等） | 180 秒 |

---

### 安装参数（params）

`params` 驱动**安装时**的表单。前端按 yaml 顺序渲染输入框。

```yaml
params:
  - key: port                        # 必需 - 表单字段 key（compose 占位符 {{port}}）
    label: 端口                       # 必需 - 显示标签
    type: number                     # 必需 - text / number / password
    default: 5432                    # 可选 - 默认值
    required: true                   # 可选 - 是否必填
    auto_generate: true              # 可选 - type=password 时自动生成 16 字符随机串
```

#### `type` 合法值

| 值 | 渲染 | 占位符行为 |
|----|------|-----------|
| `text` | 文本输入框 | 直接替换 |
| `number` | 数字输入框 | 验证 1–65535 范围（用于端口） |
| `password` | 密码输入框 | 入库前 AES 加密，敏感字段标记 |

#### 端口参数特殊行为

后端对 `key` 包含 `port` 且 `type: number` 的参数会自动：
1. 安装前检查端口是否被占用
2. 冲突时返回 `port_conflict` 错误，提供"自动改用下一个可用端口"
3. 安装期间预留端口（30 分钟），防止并发安装抢同一端口

**约定**：端口字段命名 `port` / `http_port` / `https_port` / `fpm_port` / `ftp_port` 等。

#### auto_generate 与敏感字段

```yaml
- key: db_password
  label: postgres 用户密码
  type: password
  default: ""
  required: true
  auto_generate: true                # 用户不填时自动生成
```

`auto_generate: true` 仅对 `type: password` 生效。生成 16 字符 `[a-zA-Z0-9]` 随机串。

**敏感字段会自动加密入库**（参见 backend/internal/service/app_install.go `sensitiveParamKeys`）：
- `password`
- `root_password`
- `db_password`
- `admin_password`
- `db_root_password`
- `admin_token`
- `encryption_key`

新增其他敏感 key 时需要在后端 `sensitiveParamKeys` 中加上。

---

### 挂载目录权限（volumes_init）

**关键安全点**：容器内非 root 进程（postgres UID 70、mysql 999、redis 999）无法写入宿主机 root 所有的挂载目录。声明 `volumes_init` 让面板安装时预先 chown。

```yaml
# 容器内 postgres 进程以 UID 70 运行；预先 chown 数据/日志目录避免 Permission denied
volumes_init:
  - path: /www/server/postgresql/data    # 宿主机目录路径
    uid: 70                              # 容器内进程 UID
    gid: 70                              # 容器内进程 GID
    mode: "0700"                         # 可选：chmod 八进制字符串
  - path: /www/logs/postgresql
    uid: 70
    gid: 70
```

#### 字段说明

| 字段 | 必需 | 说明 |
|------|------|------|
| `path` | ✓ | 宿主机目录绝对路径，支持 `{{key}}` 占位符（PHP 多版本场景） |
| `uid` | ✓ | 容器内进程的 UID。0 视为 root，不会执行 chown |
| `gid` | ✓ | 容器内进程的 GID |
| `mode` | optional | 八进制字符串如 `"0755"`、`"0700"`。留空保持 0755 |

#### 已知镜像的 UID

| 镜像 | UID:GID | 备注 |
|------|---------|------|
| `postgres` | 70:70 | postgres 用户 |
| `mysql` / `mariadb` | 999:999 | mysql 用户 |
| `mongo` | 999:999 | mongodb 用户 |
| `redis` | 999:1000 | redis 用户 |
| `elasticsearch` | 1000:0 | elasticsearch 用户 |
| `grafana` | 472:472 | grafana 用户 |
| `prometheus` | 65534:65534 | nobody 用户 |
| `nginx` / `httpd` / `openresty` | root（在 entrypoint 切换） | 不需要 volumes_init |

如何查 UID：

```bash
docker run --rm <image> id           # 看 ENTRYPOINT 后的实际用户
docker run --rm <image> cat /etc/passwd | grep <user>
```

#### CI 强制校验

`scripts/lint-volumes-init.go` 会在 PR 检查：**镜像在已知非 root 镜像清单（`nonRootImages` map）但未声明 `volumes_init` 时 CI 失败**。

加新非 root 镜像时同步更新 `lint-volumes-init.go` 的 `nonRootImages`。

#### 内置兜底策略（不需要声明）

任何 `/www/logs/<key>` 挂载目录如果**没有显式声明**，面板安装时会自动 chmod 0777 兜底。这是为了应对 contributor 忘了写 yaml，至少日志目录不会卡住启动。

> 不要依赖兜底策略，请显式声明。0777 是兜底不是最佳实践。

---

### 初始化文件（init_files）

用于在安装时把配置文件写入 `/www/server/<key>/`。

```yaml
init_files:
  - path: /www/server/postgresql/postgresql.conf      # 必需 - 写到哪
    file: init/postgresql.conf                         # 文件来源（与 content 二选一）
  - path: /www/server/postgresql/pg_hba.conf
    file: init/pg_hba.conf
  - path: /www/wwwroot/{{key}}/dynamic.cfg            # path 支持 {{key}} 占位符
    content: |                                         # 内联内容
      key1 = {{port}}
      key2 = {{db_password}}
```

#### 字段说明

| 字段 | 必需 | 说明 |
|------|------|------|
| `path` | ✓ | 目标路径。支持 `{{key}}` 占位符 |
| `file` | * | 相对路径，引用同级 `init/` 下的文件 |
| `content` | * | 内联内容（多行用 yaml `\|`） |

\* `file` 和 `content` 必须二选一。

#### content 中的占位符

`content`（或 `file` 引用的文件）内容会被以下占位符替换：

- `{{<param.key>}}` — 用户输入或默认值，如 `{{port}}`、`{{db_password}}`
- `{{key}}` — 应用 key

不支持 `{{tag}}` `{{version}}`（这些是 compose 模板专用）。

#### 注意

- 安装时**总是覆盖**已存在的同名文件（确保配置是最新版本）
- 这些文件的删除由卸载流程负责
- 文件权限默认 0644，无法在 yaml 配置（需要的话用 volumes_init 的 mode 改父目录权限）

---

### 动态变量（compose_vars）

某些 compose 占位符无法直接来自用户输入，需要从其他地方拼接。例如 phpMyAdmin 要连接 mysql，需要从服务注册表读 mysql 容器名。

```yaml
compose_vars:
  - placeholder: "{{pma_host}}"            # compose 中的占位符
    source: registry_password              # 数据来源
    registry_type: mysql                   # 从哪个服务读
    template: "{{username}}:{{password}}@panel_mysql:3306"   # 值的模板
    empty_value: ""                        # 服务未安装时的兜底
```

#### `source` 合法值

##### `registry_password`

从服务注册表读取已安装服务的凭据。模板里 `{{username}}` 和 `{{password}}` 会被替换。

```yaml
compose_vars:
  - placeholder: "{{wp_db_host}}"
    source: registry_password
    registry_type: mysql
    template: "panel_mysql"               # 这里只用 host，username/password 在 env 里单独传
    empty_value: ""
```

##### `param`

直接用安装参数填充。常用于条件性 compose 片段（如 redis 是否带密码）。

```yaml
# redis：用户填密码 → 命令行加 --requirepass，否则空字符串
compose_vars:
  - placeholder: "{{redis_auth_cmd}}"
    source: param
    template: " --requirepass {{password}}"
    empty_value: ""                        # 用户没填密码时
```

#### 字段说明

| 字段 | 必需 | 说明 |
|------|------|------|
| `placeholder` | ✓ | compose 中的占位符（含 `{{}}`） |
| `source` | ✓ | `registry_password` 或 `param` |
| `registry_type` | source=registry_password 时 | 从哪类服务读（mysql/postgresql/mongodb） |
| `template` | ✓ | 值的模板，可包含 `{{username}}`、`{{password}}` 或参数 key |
| `empty_value` | optional | 关键参数为空时的兜底（默认空串） |

---

### 服务依赖（dependencies / service_dependencies）

```yaml
# 显式依赖：必须先安装某个具体应用
dependencies:
  - mysql                              # 这个应用安装前 mysql 必须存在

# 服务类型依赖：必须先安装"任意 webserver/php/database"
service_dependencies:
  - type: webserver                    # 必须先有 webserver
    label: "Web 服务器（Nginx/OpenResty/Apache）"
  - type: php
    label: "PHP 运行时"
```

#### `dependencies` 字段

按 app key 列出。安装时检查每个 key 是否已安装，缺失时返回 `dependency_missing` 错误，提供一键安装依赖。

#### `service_dependencies` 字段

按服务类型列出。只要有任意一个该类型的服务已安装即可。常用于 wordpress 这类只关心"有没有 web server"的应用。

| `type` | 满足条件 |
|--------|---------|
| `webserver` | 已安装 nginx/openresty/apache 任一 |
| `php` | 已安装 php_74/80/81/82/83 任一 |
| `database` | 已安装 mysql/postgresql/mongodb/redis 任一（细分时按 db_type） |

---

### 资源限制（resource_limits）

```yaml
resource_limits:
  memory_mb: 512                       # 内存上限 MB
  cpus: 1.0                            # CPU 上限（核数）
  mem_reserve_mb: 256                  # 内存软限制（推荐最低）
```

值会通过 `{{mem_limit}}` `{{cpu_limit}}` `{{mem_reserve}}` 注入 compose 模板。

**用户也能在前端覆盖这些值**（安装抽屉 → 高级 → 资源限制）。yaml 提供的是合理默认。

未声明 `resource_limits` 时，compose 中的 `deploy.resources` 块会被整段移除（容器无限制）。

| 应用类型 | 推荐 memory_mb | cpus |
|---------|--------------|------|
| 数据库（mysql/postgres） | 512 | 1.0 |
| 数据库（mongodb） | 768 | 1.0 |
| Redis | 256 | 0.5 |
| Web 服务器（nginx） | 256 | 0.5 |
| PHP-FPM | 512 | 1.0 |
| 工具（phpmyadmin/pgweb） | 128 | 0.5 |

---

### 反代网站（proxy_spec）

声明应用支持创建反代网站（用域名访问 phpMyAdmin、pgweb 等带 web UI 的工具）。

```yaml
proxy_spec:
  default_port: 8080                   # 反代目标端口（与 port 参数对应）
  port_param: port                     # 优先从此安装参数读端口（高于 default_port）
  ws_support: false                    # 是否默认开启 WebSocket
  description: "通过域名访问 phpMyAdmin，支持 HTTPS"
```

声明后，应用安装抽屉会出现"开启反代"开关；管理抽屉会有"反代网站"模块。

---

### 管理抽屉（manage / actions / tabs / php_extensions）

#### `manage` —— 配置管理

驱动管理抽屉的"配置"tab。每个条目对应一个表单项，写入 `file_path` 指定的配置文件。

```yaml
manage:
  - key: max_connections                                # 配置项 key（写入文件时用）
    label: 最大连接数                                    # 显示标签
    type: number                                         # text/number/select/switch/textarea
    description: "每连接约 5–10MB"                       # 帮助文本
    options:                                             # type=select 时
      - { value: "256MB", label: "256MB（推荐 2GB+）" }
      - { value: "512MB", label: "512MB（推荐 4GB+）" }
    restart_required: true                               # 修改后需要重启
    file_path: /www/server/postgresql/postgresql.conf    # 写到哪
    ini_section: memory                                  # INI section（PHP 用）
    format: kv_eq                                        # 配置文件格式（见下方说明）
    bool_format: "on/off"                                # type=switch 时必填（见下方说明）
```

##### `format` 字段

控制配置文件的解析和写入方式：

| 值 | 格式 | 适用文件 |
|----|------|---------|
| `""` / `ini`（默认） | `key = value`，支持 `[section]` | php.ini、my.cnf |
| `kv_eq` | `key = value`，无 section | postgresql.conf |
| `kv_space` | `key value`，空格分隔，支持行尾分号 | nginx.conf、redis.conf、pure-ftpd.conf、httpd.conf |
| `env` | `KEY=value`，环境变量格式 | .env 文件 |

> **注意**：`nginx.conf` 和 `openresty.conf` 使用 `key value;` 格式（分号结尾），属于 `kv_space`。面板会自动保留行尾分号，确保写入后的文件格式正确。

##### `ini_section` 字段

指定 INI 格式（`format: ""` 或 `format: ini`）中的 section 名称：
- `format: ini`（php.ini、my.cnf）→ 填写对应 section，如 `PHP`、`opcache`、`mysqld`
- `format: kv_space`（nginx.conf、redis.conf 等）→ **必须填 `""`**，kv_space 不区分 section
- `format: kv_eq`（postgresql.conf）→ **必须填 `""`**，无 section
- `format: env`（.env 文件）→ **必须填 `""`**，无 section

```yaml
# ✓ PHP（INI 格式，有 section）
- key: max_execution_time
  format: ""              # 省略也可以，默认 ini
  ini_section: PHP        # [PHP] section

# ✓ Nginx（kv_space 格式，无 section）
- key: worker_processes
  format: kv_space
  ini_section: ""         # 必须空，kv_space 不使用 section

# ✓ PostgreSQL（kv_eq 格式，无 section）
- key: max_connections
  format: kv_eq
  ini_section: ""         # 必须空

# ✗ 错误：kv_space 格式填了 section（会被忽略，造成误导）
- key: gzip
  format: kv_space
  ini_section: http       # ← 错误，kv_space 不支持 section
```

##### `bool_format` 字段（`type: switch` 必填）

**所有 `type: switch` 的字段都必须声明 `bool_format`。**

前端统一传 `"true"`/`"false"`，后端写入文件时按 `bool_format` 转换为目标格式。不声明时系统会尝试从文件中已有的值推断，但推断在以下情况会失败：
- 文件被写坏（如 `KeepAlive false` 而不是 `KeepAlive On`）
- key 在文件中不存在（新增配置项）

| `bool_format` 值 | 写入格式 | 适用应用 |
|-----------------|---------|---------|
| `"On/Off"` | `On` / `Off` | Apache httpd.conf |
| `"on/off"` | `on` / `off` | Nginx/OpenResty nginx.conf、PostgreSQL postgresql.conf |
| `"yes/no"` | `yes` / `no` | Redis redis.conf、Pure-FTPd pure-ftpd.conf |
| `"YES/NO"` | `YES` / `NO` | 全大写 yes/no |
| `"1/0"` | `1` / `0` | MySQL my.cnf（slow_query_log） |
| `"0/1"` | `0` / `1` | 同上，语义相同（true=1 false=0） |
| `"On/Off"` | `On` / `Off` | PHP php.ini（display_errors 等） |

格式为 `"真值/假值"`，前端 `true` 写入真值，`false` 写入假值。

**示例：**

```yaml
# Apache：KeepAlive 用 On/Off
- key: KeepAlive
  type: switch
  bool_format: "On/Off"
  format: kv_space
  file_path: /www/server/apache/httpd.conf

# Nginx：gzip 用 on/off
- key: gzip
  type: switch
  bool_format: "on/off"
  format: kv_space        # nginx.conf 是 kv_space 格式
  ini_section: ""
  file_path: /www/server/nginx/nginx.conf

# Redis：appendonly 用 yes/no
- key: appendonly
  type: switch
  bool_format: "yes/no"
  format: kv_space
  ini_section: ""
  file_path: /www/server/redis/redis.conf

# MySQL：slow_query_log 用 1/0
- key: slow_query_log
  type: switch
  bool_format: "1/0"
  file_path: /www/server/mysql/my.cnf
  ini_section: mysqld

# PHP：display_errors 用 On/Off，opcache.enable 用 1/0
- key: display_errors
  type: switch
  bool_format: "On/Off"
  file_path: /www/server/php_82/php.ini
  ini_section: PHP

- key: opcache.enable
  type: switch
  bool_format: "1/0"
  file_path: /www/server/php_82/php.ini
  ini_section: opcache
```

> **为什么不自动推断？** 推断依赖文件中已有的值。一旦文件被写坏（如历史 bug 写入了 `KeepAlive false`），推断就会永久失效，形成无法自愈的状态。显式声明是唯一可靠的方式。

#### `actions` —— 操作按钮

驱动管理抽屉的"操作"tab。

```yaml
actions:
  - id: reload                              # 内部 ID
    label: 重载配置                         # 显示
    icon: refresh                           # 图标 key
    confirm: false                          # 点击是否需要确认弹窗
    danger: false                           # 危险操作（红色按钮）
    exec: "psql -U postgres -c 'SELECT pg_reload_conf();'"   # 容器内执行的命令
```

`exec` 由后端在容器内 `docker exec` 执行。`{{container}}` 占位符会被替换为容器名。

或用 API 路径调用前端定义的 handler：

```yaml
  - id: backup_now
    label: 立即备份
    api_path: /api/v1/apps/postgresql/backup
```

`exec` 和 `api_path` 二选一。

#### `tabs` —— Tab 顺序

```yaml
tabs:
  - id: overview
    label: 概览
    type: overview
  - id: config
    label: 配置
    type: config_form
  - id: extensions
    label: 扩展
    type: extensions
```

`type` 合法值：`overview` / `config_form` / `config_file` / `extensions` / `logs` / `performance` / `backup` / `upgrade` / `actions`。

未声明 `tabs` 时使用默认 tab 集合。

#### `php_extensions` —— PHP 扩展

仅 PHP 应用使用。驱动 PHP 管理抽屉的"扩展"tab。

```yaml
php_extensions:
  - name: gd
    label: GD 图形库
    install_command: "docker-php-ext-install gd"
```

---

### 安装引导与提示

```yaml
install_hints:                          # 安装完成时显示的提示（多行）
  - "数据库管理 → 可以创建和管理 PostgreSQL 数据库"
  - "计划任务 → 可以设置数据库自动备份"

install_guide:                          # 安装成功页的引导卡片
  description: "已自动注册为默认 PostgreSQL 服务"
  features:
    - "数据库管理 - 创建和管理 PostgreSQL 数据库"
  primary_action:
    label: "前往数据库管理"
    route: "/database"
```

---

### 数据库自动初始化（db_init）

声明该应用安装后需要自动在某个数据库中创建库和用户（典型场景：WordPress 安装时在 MySQL 中创建 wp 用户和库）。

```yaml
db_init:
  registry_type: mysql                  # 在哪个数据库创建
  db_name_param: db_name                # 从安装参数读取库名
  db_user_param: db_user                # 用户名
  db_password_param: db_password        # 密码
```

---

### 数据库 dump（db_dump）

多容器应用（如 WordPress 自带 MySQL）的备份/恢复配置：

```yaml
db_dump:
  container_suffix: mysql               # 数据库容器名后缀（panel_<container_suffix>）
  db_name: wordpress                    # 库名
  db_user: wp_user
  password_config_key: db_password      # 从安装配置读密码
```

---

### 其他字段

```yaml
fixed_port: 9000                        # 固定端口（PHP-FPM 等无 params.port 字段的应用）
has_web_ui: true                        # 是否有 Web UI（影响首页"打开"按钮）
version_key_template: "php_{{version_nodot}}"   # 多版本应用的 key 模板
reload_cmd: "nginx -s reload"           # 配置热重载命令（默认按 service_type 选）
```

---

## docker-compose.yml 模板规范

### 基本骨架

```yaml
services:
  <app-key>:
    image: <image>:{{tag}}              # 必需 - 用 {{tag}} 占位符
    container_name: panel_<app-key>     # 必需 - 容器名规范：panel_<key>
    restart: unless-stopped              # 推荐
    ports:
      - "{{port}}:<container-port>"     # 推荐：用 {{<param-key>}} 引用 params
    environment:
      <KEY>: "{{<param-key>}}"          # 引用 params 中的密码、配置等
    volumes:
      - /www/server/<app-key>/data:/var/lib/<app>/data
      - /www/server/<app-key>/<conf>:/etc/<app>/<conf>:ro
      - /www/logs/<app-key>:/var/log/<app>
    networks:
      - panel_network                   # 必需 - 加入 panel_network
    deploy:
      resources:
        limits:
          memory: {{mem_limit}}
          cpus: "{{cpu_limit}}"
        reservations:
          memory: {{mem_reserve}}

networks:
  panel_network:
    external: true                      # 必需 - 用面板预创建的网络
    name: panel_network
```

### 容器命名

固定格式 `panel_<key>`。后端依赖这个命名规则做容器查找、日志读取、统计采集等。**不要**改成其他格式。

### 网络

**应用的网络模式由其功能类型决定，规则如下：**

#### 使用 `network_mode: host` 的应用

下列应用类型**必须**使用 host 网络模式，**不得**使用 `panel_network`：

| 类型 | 应用 | 原因 |
|------|------|------|
| **Web 服务器** | nginx, openresty, apache | 直接监听宿主机 80/443，端口转发最小延迟 |
| **PHP-FPM** | php_74, php_80, php_81, php_82, php_83 | PHP 代码需要用 `localhost`/`127.0.0.1` 连接 MySQL/Redis（与宝塔/1Panel 行为一致） |
| **FTP 服务器** | pureftpd | 被动模式（PASV）必须通告宿主机真实 IP，bridge 网络下 PUBLICHOST 无法正确工作 |

```yaml
# ✓ PHP/Nginx/Apache/FTP 的正确写法
services:
  php:
    image: jzpanel/php:{{tag}}
    container_name: panel_php_82
    restart: unless-stopped
    network_mode: host          # ← 必须 host，不能是 panel_network
    volumes:
      - ...
    # ← 无 networks 块，无 ports 块（host 模式下 FPM 直接绑宿主机端口）
```

> **为什么 PHP 要用 host 模式？**
>
> PHP 容器在 `panel_network`（bridge 网络）时，`localhost` 和 `127.0.0.1` 指向的是 PHP 容器自己的 loopback，无法连接到 MySQL 容器。使用 host 网络后，PHP 进程直接绑在宿主机网络栈上，`localhost:3306` 就等于访问宿主机 3306 端口，而 MySQL 容器已通过 `ports: "{{port}}:3306"` 暴露到宿主机 3306，因此 PHP 用 `localhost` 连接 MySQL 完全正常——和宝塔、1Panel 原生安装的行为一致。

#### 使用 `panel_network` 的应用

下列应用类型使用 `panel_network` + 端口映射：

| 类型 | 应用 | 原因 |
|------|------|------|
| **数据库** | mysql, postgresql, mongodb, redis | 数据不需要用 localhost 连，面板通过宿主机端口管理；数据库之间通过容器名互连 |
| **管理工具** | phpmyadmin, pgweb, mongoexpress | 需要通过容器名（`panel_mysql` 等）连接数据库，必须在同一网络 |

```yaml
# ✓ 数据库/管理工具的正确写法
services:
  mysql:
    image: mysql:{{tag}}
    container_name: panel_mysql
    restart: unless-stopped
    ports:
      - "{{port}}:3306"         # ← 宿主机端口映射（面板管理用）
    networks:
      - panel_network           # ← 必须 panel_network（让工具类容器能通过容器名连接）

networks:
  panel_network:
    external: true
    name: panel_network
```

#### 判断用哪种网络模式

```
新应用需要哪种网络模式？

├─ 用户代码（PHP/Python/Node）需要用 localhost/127.0.0.1 连接它？
│   → host 模式（如 PHP-FPM、FTP）
│
├─ 它是 Web 服务器，需要监听 80/443？
│   → host 模式（Nginx/Apache/OpenResty）
│
├─ 它需要连接数据库容器（panel_mysql 等）？
│   → panel_network（如 phpMyAdmin、pgweb、Mongo Express）
│
└─ 它是数据库，需要被面板管理且被工具类容器连接？
    → panel_network + ports 映射（MySQL/PostgreSQL/Redis/MongoDB）
```

> ⚠️ **常见错误**：把 PHP-FPM 放进 `panel_network` 会导致用户在 WordPress/Laravel 等源码的数据库配置里填 `localhost` 或 `127.0.0.1` 时连接失败。这是一个不易发现的问题，因为 PHP 容器本身运行正常，只有用户实际配置应用时才会报错。

### 路径约定

| 路径前缀 | 用途 | 备注 |
|---------|------|------|
| `/www/server/<key>/` | 数据 + 配置 | data/、conf/、各种 .conf |
| `/www/logs/<key>/` | 日志 | 兜底 chmod 0777 |
| `/www/wwwroot/<key>/` | 网站文件（CMS 类） | 仅 WordPress 等用 |
| `/www/backup/apps/<key>/` | 备份文件 | 由备份模块管理，yaml 不直接挂载 |

这些路径会通过 `renderComposeTemplate` 自动按 `paths.Global` 替换为用户配置的实际路径。新应用**不要硬编码**别的路径。

### 端口暴露

所有需要从外部访问的端口必须用 `{{<param-key>}}` 占位符，与 `params` 中 `key` 一致。这样面板能：
- 安装时检测端口冲突
- 卸载时释放端口预留
- 自动同步防火墙规则

```yaml
# ✓ 正确
ports:
  - "{{http_port}}:80"
  - "{{https_port}}:443"

# 错误：硬编码端口（除非确实没法 param 化，如 panel_mongodb_70 自动注入）
ports:
  - "27017:27017"
```

---

## 模板占位符（Placeholders）

### 内置占位符（自动替换）

| 占位符 | 含义 | 来源 |
|-------|------|------|
| `{{tag}}` | 当前安装版本的真实 tag | versions[].tag |
| `{{version}}` | 当前安装版本的 display（向后兼容） | versions[].display |
| `{{digest}}` | 镜像 digest，含 `@` 前缀 | versions[].digest（声明时） |
| `{{version_nodot}}` | display 去除 `.`（如 8.2 → 82） | 计算得到 |
| `{{key}}` | 应用 key | app.yaml.key |
| `{{mem_limit}}` | 内存限制（如 `512m`） | resource_limits.memory_mb |
| `{{cpu_limit}}` | CPU 限制（如 `1.0`） | resource_limits.cpus |
| `{{mem_reserve}}` | 内存软限制 | resource_limits.mem_reserve_mb |
| `{{<param-key>}}` | 任意安装参数 | params[].key |
| `{{<placeholder>}}` | compose_vars 声明的占位符 | compose_vars[].placeholder |

### 占位符的字符串转义

`params` 中字符串类型的值（特别是 password）会经过 YAML 转义，处理特殊字符如 `'` `"` `\`。环境变量赋值要用引号包裹：

```yaml
# ✓ 正确
environment:
  POSTGRES_PASSWORD: "{{db_password}}"

# 不推荐：密码含 # @ 等会出问题
environment:
  POSTGRES_PASSWORD: {{db_password}}
```

### 路径占位符（自动替换）

`renderComposeTemplate` 会把以下路径前缀替换为 `paths.Global` 的实际值：

```yaml
/www/wwwroot       → paths.Global.WWWRoot
/www/server        → paths.Global.Server
/www/backup        → paths.Global.Backup
/www/logs          → paths.Global.Logs
panel_             → paths.Global.ContainerPrefix
```

新应用**始终用** `/www/...` 写法，不要写绝对路径或 `${PANEL_PATH}/...`。

---

## 挂载目录权限（volumes_init）

> 已在 [app.yaml 字段参考 → volumes_init](#挂载目录权限volumes_init-1) 中详述。这里给出**为何如此设计**和实践经验。

### 为什么需要

容器内进程的运行 UID 由镜像 ENTRYPOINT 决定，不是 docker 默认的 root。常见非 root 镜像：

```
postgres → UID 70
mysql    → UID 999
mongo    → UID 999
redis    → UID 999
```

宿主机挂载目录默认是 `root:root 0755`。容器内非 root 进程对这些目录**没有写权限**，会报：

```
FATAL: could not open log file "/var/log/postgresql/postgresql-2026-05-25.log": Permission denied
```

### 必须显式声明的目录

任何**容器内进程要写入的挂载目录**都必须声明 `volumes_init`：

```yaml
volumes_init:
  # 数据目录：必须 chown，且通常 mode 0700
  - path: /www/server/<app>/data
    uid: <container-uid>
    gid: <container-gid>
    mode: "0700"

  # 日志目录：可 chown，也可什么都不写（兜底 0777）
  - path: /www/logs/<app>
    uid: <container-uid>
    gid: <container-gid>

  # 配置目录：通常不写入（:ro 挂载），不需要 chown
```

### 不需要声明的目录

- `:ro` 只读挂载（如 `:/etc/<app>/conf:ro`）
- 容器内 root 镜像（nginx/httpd/openresty 在 entrypoint 内切换）
- 容器自管理的目录（如 docker volume `mysql_data`）

---

## 初始化文件（init_files）

> 详见 [app.yaml → init_files](#初始化文件init_files-1)。

实践提示：

1. **优先用 `file:` 引用外部文件**，不要内联 YAML。配置文件容易长且含特殊字符，外部文件可以单独编辑、有 syntax highlight。

2. **路径必须与 compose 中的挂载源对应**：

   ```yaml
   # init_files 写到这里
   - path: /www/server/postgresql/postgresql.conf

   # docker-compose.yml 挂载这里
   volumes:
     - /www/server/postgresql/postgresql.conf:/etc/postgresql/postgresql.conf:ro
   ```

3. **占位符可在 content/file 内容中使用**：

   ```ini
   # postgresql.conf
   port = {{port}}
   max_connections = 100
   ```

   会被替换为用户输入的端口。

---

## 动态变量（compose_vars）

> 详见 [app.yaml → compose_vars](#动态变量compose_vars-1)。

### 实战示例

#### 示例 1：phpMyAdmin 连接 MySQL

```yaml
# app.yaml
compose_vars:
  - placeholder: "{{pma_host}}"
    source: registry_password
    registry_type: mysql
    template: "panel_mysql"             # 用容器名
    empty_value: ""

# docker-compose.yml
environment:
  PMA_HOST: "{{pma_host}}"              # 解析后变成 panel_mysql
```

#### 示例 2：Redis 可选密码

```yaml
# app.yaml
params:
  - key: password
    type: password
    auto_generate: false                 # 用户可以填空表示不要密码
compose_vars:
  - placeholder: "{{redis_auth_cmd}}"
    source: param
    template: " --requirepass {{password}}"
    empty_value: ""

# docker-compose.yml
command: redis-server /etc/redis/redis.conf{{redis_auth_cmd}}
# 用户填密码 → command: redis-server /etc/redis/redis.conf --requirepass abc123
# 用户没填 → command: redis-server /etc/redis/redis.conf
```

#### 示例 3：Mongo Express 连接字符串

```yaml
compose_vars:
  - placeholder: "{{me_mongodb_url}}"
    source: registry_password
    registry_type: mongodb
    template: "mongodb://{{username}}:{{password}}@panel_mongodb:27017/?authSource=admin"
    empty_value: "mongodb://localhost:27017"   # mongodb 未安装时的兜底

# docker-compose.yml
environment:
  ME_CONFIG_MONGODB_URL: "{{me_mongodb_url}}"
```

---

## 服务注册与依赖

### 自动注册到服务注册表

`service_type` 不为空的应用会在安装成功后**自动注册**到服务注册表。其他应用（如 phpMyAdmin）通过 `compose_vars` 的 `source: registry_password` 拉取已注册服务的容器名/端口/凭据。

注册的字段：
- `key`：app key
- `name`：app name
- `type`：service_type
- `db_type`：db_type
- `internal_host`：容器名（默认 `panel_<key>`）
- `external_port`：从 params 中提取的端口
- `username` / `password`：从 params 中提取（按命名约定 `*_user` `*_password`）

### 卸载检查

`service_type: webserver` / `database` 卸载时检查是否有其他对象（网站/数据库）依赖此服务，有则拒绝。

---

## icon 与图标

```
<app-key>/icon.svg            # 必需：SVG 矢量图，128×128 viewbox
<app-key>/icon.png            # 备用 PNG，128×128
```

要求：
- 优先 SVG（缩放清晰、文件小、可主题色）
- 有官方 logo 用官方 logo（注意版权，仅用于应用商店展示属合理使用）
- 没有官方 logo 的工具类应用，可用首字母图标
- 风格统一：圆角、扁平、单色或双色

---

## CI 校验

push 到 main 或 PR 时，GitHub Actions 跑两个 check：

### 1. `validate.yml` — `check-versions.go`

校验每个 yaml 中的 `versions[]`：
- 每个 `tag` 在 `image` 镜像下真实存在（HEAD `/v2/<image>/manifests/<tag>`）
- `architectures` 声明的每个架构都能拉到 manifest（multi-arch 镜像）
- `digest` 字段（如声明）与真实 digest 一致
- `rolling: true` 跳过 manifest 校验

`--strict` 模式（cron-validate.yml 每天跑）：
- `deprecated: true` 也校验（捕获上游主动删 tag）
- 失败时自动开 issue

### 2. `validate.yml` — `lint-volumes-init.go`

校验非 root 镜像应用必须声明 `volumes_init`：
- 镜像在 `nonRootImages` 清单中
- 但 yaml 没有 `volumes_init` 字段
- → CI 失败，PR 无法 merge

### 本地预检

PR 前在本地跑：

```bash
cd apps-repo/scripts
go run check-versions.go --apps-dir ..
go run lint-volumes-init.go --apps-dir ..
```

---

## 新增应用的标准流程

### Step 1：调研

- 找到上游官方 Docker 镜像
- 确认镜像支持的架构（`docker manifest inspect <image>`）
- 确认容器内进程 UID（`docker run --rm <image> id`）
- 确认镜像默认配置文件路径
- 确认日志输出方式（stderr 还是文件）

### Step 2：创建目录结构

```bash
mkdir -p apps-repo/<key>/init
cd apps-repo/<key>
touch app.yaml docker-compose.yml
# 复制官方默认配置到 init/
```

### Step 3：写 app.yaml

参考已有应用（mysql/postgresql/redis 是好模板）。**必填字段**：
- key, name, category, description
- versions（至少一个 entry，含 image/tag/architectures）
- default_version
- service_type（如适用）
- params（端口、密码等）

**强烈建议**：
- volumes_init（如果是非 root 镜像）
- init_files（如果有配置文件）
- resource_limits（合理默认值）

### Step 4：写 docker-compose.yml

参考已有应用。**必含**：
- `image: <image>:{{tag}}`
- `container_name: panel_<key>`
- `networks: [panel_network]` 加 `external: true`
- `deploy.resources` 块（即使 yaml 没声明 limits，模板里也写）

### Step 5：放图标

```bash
# 优先 SVG
curl https://example.com/logo.svg -o icon.svg
# 或截屏后转 PNG，128×128
```

### Step 6：本地校验

```bash
cd apps-repo/scripts
go run check-versions.go --apps-dir .. --app <key>
go run lint-volumes-init.go --apps-dir ..
```

### Step 7：本地端到端测试

在 dev 服务器上：

```bash
# 同步本地 yaml 到 panel apps 目录
rsync -av apps-repo/<key>/ /www/server/panel/apps/<key>/

# 在面板里点"重新扫描本地应用"
# 或用 API
curl -X POST http://localhost:8080/api/v1/apps/reload-registry

# 在面板里安装该应用，确认：
# - 健康检查通过
# - 容器 running
# - 数据/日志目录权限正确
# - 配置文件被正确写入
# - 卸载能完整清理
```

### Step 8：开 PR

```bash
git checkout -b add-<key>-app
git add apps-repo/<key>/
git commit -m "feat(<key>): add <Name> app"
git push origin add-<key>-app
```

PR 描述应包含：
- 应用简介
- 测试过的版本（display/tag）
- 测试过的架构（amd64/arm64）
- 是否需要 `volumes_init`，UID 是多少

### Step 9：CI 通过 + 维护者 review → merge

merge 后 `bump.yml` 自动 bump version.json，jsDelivr cache purge。约 5 分钟内所有装了面板的用户能在"应用商店"看到新应用。

---

## 常见错误与排查

### 1. 容器反复重启 `Permission denied`

**原因**：忘了声明 `volumes_init`，容器内非 root 用户无法写挂载目录。

**修复**：参考 [挂载目录权限](#挂载目录权限volumes_init)。

### 2. `image_tag_not_found`

**原因**：yaml 写的 tag 在 Docker Hub 上不存在或被删了。

**修复**：去 hub.docker.com 查实际 tag，更新 yaml。CI 应该已经拦住，merge 前发生说明 CI 没启用或 tag 之后被上游删了。

### 3. `port_conflict`

**原因**：用户的 `port` 参数撞了已用端口。

**预防**：选合理的默认端口。常用端口先到先得（80、443、3306、5432 等已被占用）。新应用建议默认端口避开 1024–9999 常见区间，用 8000–9000 或 10000+。

### 4. 安装日志显示 `manifest unknown`

**原因**：yaml 的 `image` 字段写错了（如 `postgresql` 而不是 `postgres`）。

**修复**：去 hub.docker.com 搜准确的 image 名。

### 5. compose 占位符未替换（容器报 `{{port}} is not a valid integer`）

**原因**：`{{xxx}}` 占位符在 yaml 中拼错或 `params` 没声明 `xxx`。

**修复**：检查 compose 中所有 `{{}}` 是否都有对应的 `params` key 或 `compose_vars`。

### 6. 健康检查超时但容器其实 running

**原因**：`health_check_timeout` 太短（默认 120 秒）。

**修复**：在 yaml 加 `health_check_timeout: 180`。

### 7. PHP 等多版本应用的 `version_key_template`

PHP 7.4 的 app key 是 `php_74`，不是 `php`。声明 `version_key_template: "php_{{version_nodot}}"` 让前端 UI 能识别这是同一应用的多版本。

### 8. switch 字段写入了 `true`/`false` 而不是 `On`/`Off`

**原因**：`type: switch` 字段没有声明 `bool_format`，系统尝试从文件中已有的值推断格式，但推断失败（文件被写坏、key 不存在等）。

**症状**：Apache 报 `KeepAlive must be On or Off`，Nginx 报 `invalid value "true" in ... directive`，容器反复重启。

**修复**：在 `app.yaml` 的 switch 字段加上 `bool_format`，参考 [manage → bool_format 字段](#bool_format-字段type-switch-必填)。

**预防**：所有 `type: switch` 字段都必须声明 `bool_format`，不要依赖自动推断。

### 9. PHP 用 `localhost` 连不上 MySQL（`SQLSTATE[HY000] [2002] Connection refused`）

**原因**：PHP 容器放在 `panel_network`（bridge 网络）里，`localhost` 和 `127.0.0.1` 指向的是 PHP 容器自己的 loopback，而不是宿主机。MySQL 容器在另一个 bridge 容器里，两者无法通过 `localhost` 互通。

**症状**：
- WordPress 安装时"数据库连接建立错误"
- Laravel `.env` 里 `DB_HOST=127.0.0.1` 报 `Connection refused`
- PHP 代码里 `new PDO('mysql:host=localhost;...')` 失败

**修复**：PHP 容器改为 `network_mode: host`（参见 [docker-compose.yml 模板规范 → 网络](#网络) 章节）。

**预防**：新增 PHP 版本或任何运行 PHP-FPM 的容器，**必须使用 host 网络模式**，同时在 `php-fpm.conf` 的 `listen` 字段填写该版本对应的专属端口（9074/9080/9081/9082/9083），**不能用 `0.0.0.0:9000`**（host 模式下 9000 只能被一个版本占用，多版本会冲突）。

```ini
; ✓ php_80/init/php-fpm.conf
listen = 0.0.0.0:9080

; ✓ php_82/init/php-fpm.conf
listen = 0.0.0.0:9082

; ✗ 错误：所有版本都用 9000 会冲突（host 模式）
listen = 0.0.0.0:9000
```

同样的问题也存在于 FTP 服务器（pureftpd）——PASV 被动模式在 bridge 网络下无法正常通告宿主机 IP，也必须用 host 模式。

### 10. PHP `fpm_port` 参数已移除，不要在 PHP 应用里加端口 param

**背景**：PHP 改为 host 网络模式后，FPM 监听端口由 `php-fpm.conf` 的 `listen` 字段硬编码决定，`params` 里的 `fpm_port` 不再对实际端口产生影响。已于 v1.3 移除。

**正确做法**：PHP 应用的 `params: []`（空列表），端口从版本号映射表（`registerPHP` 的 `portMap`）自动获取。

### 11. `manage` 配置项必须有 `file_path`，否则保存完全无效

**原因**：`manage` 配置项写入配置文件依赖 `file_path` 字段。缺少时面板 UI 看起来正常，但点"保存"不会写入任何文件。

**常见错误**：MongoDB 的 `mongod.conf` 是 YAML 格式，不是 INI/kv 格式，面板的写入功能**不支持 YAML 格式的配置文件**。不要为 MongoDB 写 `manage` 配置项，应引导用户通过"配置文件" tab 手动编辑。

**预防**：每个 manage 配置项检查清单：
1. 有 `file_path`？
2. `format` 与目标文件格式匹配？（INI/kv_eq/kv_space/env）
3. 如果是 `type: switch`，有 `bool_format`？

### 12. MySQL `innodb_log_file_size` 在 MySQL 8.0.30+ 已移除

**问题**：`my.cnf` 里声明 `innodb_log_file_size` 会导致 MySQL 8.4 启动失败（unknown variable）。

**修复**：从 `my.cnf` 和 `app.yaml manage` 中删除此参数。MySQL 8.4 使用 `innodb_redo_log_capacity` 替代（默认值通常足够，无需显式配置）。

### 13. `docker-compose.yml` 挂载的文件必须在 `init_files` 里声明

**问题**：compose 里挂载了 `config.user.inc.php`，但 `app.yaml` 里没有对应的 `init_files` 创建这个文件。Docker 挂载不存在的文件时会**自动创建一个同名目录**，导致容器启动失败（期望文件，得到目录）。

**规则**：compose volumes 里每一个**非目录**的宿主机挂载路径，都必须在 `init_files` 里有对应条目（或者在安装流程中通过其他方式预先创建）。

### 14. `default_version` 不应使用 rolling tag

**问题**：`default_version: "latest"` 且对应版本的 `rolling: true`。生产环境中 latest tag 随上游更新而变化，每次安装可能拉到不同版本，导致环境不一致。

**规则**：`default_version` 必须指向一个**精确版本**（有固定 tag，非 rolling）。rolling tag 可以保留在版本列表里供用户选择，但不能作为默认。

**修复**：将 `default_version` 改为最近的稳定版本，如 `"0.15"`（pgweb）。

### 15. EOL 版本应标记 `deprecated: true`

已过官方支持生命周期的版本应标记 `deprecated: true`，前端下拉里会显示警告，CI strict 模式会校验。

参考 EOL 时间线：
- PostgreSQL 14 → 2024-11 EOL → 标 deprecated
- MySQL 5.7 → 2023-10 EOL → 标 deprecated（已标）
- MongoDB 5.0 → 2024-10 EOL → 标 deprecated（已标）
- PHP 7.4 / 8.0 → 已 EOL → 标 deprecated（已标）

---



### A. 多容器应用

WordPress 自带 MySQL 容器，docker-compose.yml 里有 2 个 service。注意：
- 两个 container_name 都用 `panel_<key>_<service>` 格式
- 端口检查会自动检测所有 service 的 ports
- 卸载时两个容器都会清理
- 备份要写 `db_dump`

### B. 自定义 reload 命令

默认行为：
- `service_type: webserver` / `php` → `kill -HUP 1`
- `service_type: database` → `docker restart`
- 其他 → `docker restart`

特殊应用（如 OpenResty）需要自己声明：

```yaml
reload_cmd: "openresty -s reload"
```

### C. AI 工具集成

应用通过 yaml 暴露的 `actions` 可以被 AI 助手通过 `execute_action` 工具调用。要 AI 友好的 action：
- `id` 用动宾结构（reload、show_connections、flush_cache）
- `label` 用中文，AI 在系统提示词里能看到
- 危险操作必须 `confirm: true` 和 `danger: true`

---

## 附录 A：完整 yaml 字段速查表

| 字段 | 类型 | 必需 | 说明 |
|------|------|------|------|
| `key` | string | ✓ | 应用 key |
| `name` | string | ✓ | 显示名 |
| `category` | string | ✓ | runtime/database/tool/service/cms |
| `description` | string | ✓ | 简介 |
| `source` | string | 推荐 | 应用来源标记 |
| `icon` | string | ✓ | 图标文件名 |
| `versions[]` | list | ✓ | 版本列表 |
| `default_version` | string | ✓ | 默认版本 display |
| `architectures[]` | list | 推荐 | 应用级架构 fallback |
| `service_type` | string | optional | webserver/database/php/tool/service |
| `db_type` | string | service_type=database | mysql/pgsql/mongodb/redis |
| `health_check_timeout` | int | optional | 秒，默认 120 |
| `fixed_port` | int | optional | 固定端口（PHP-FPM 等） |
| `has_web_ui` | bool | optional | 是否有 Web UI |
| `version_key_template` | string | optional | 多版本应用的 key 模板 |
| `reload_cmd` | string | optional | reload 命令覆盖 |
| `params[]` | list | ✓（一般都需要） | 安装表单 |
| `manage[]` | list | optional | 管理配置项（含 format、bool_format 子字段） |
| `actions[]` | list | optional | 操作按钮 |
| `tabs[]` | list | optional | tab 顺序 |
| `php_extensions[]` | list | PHP only | PHP 扩展 |
| `init_files[]` | list | optional | 初始化文件 |
| `volumes_init[]` | list | 非 root 镜像必需 | 挂载目录 chown |
| `compose_vars[]` | list | optional | 动态占位符 |
| `dependencies[]` | list | optional | 显式依赖 app key |
| `service_dependencies[]` | list | optional | 服务类型依赖 |
| `resource_limits` | object | optional | CPU/内存默认 |
| `proxy_spec` | object | optional | 反代支持声明 |
| `db_init` | object | optional | 自动建库建用户 |
| `db_dump` | object | optional | 多容器备份配置 |
| `install_hints[]` | list | optional | 安装完成提示 |
| `install_guide` | object | optional | 安装成功引导卡片 |

## 附录 B：参考实现

最佳实践参考（按复杂度递增）：

1. **简单 web 服务器**：[`nginx/`](../nginx/) — 单容器，无密码，无非 root UID 问题
2. **数据库（推荐模板）**：[`postgresql/`](../postgresql/) — 完整 init_files + volumes_init + manage
3. **多版本 PHP 运行时**：[`php_83/`](../php_83/) — host 网络模式 + 固定端口 + params 为空
4. **依赖其他服务的工具**：[`phpmyadmin/`](../phpmyadmin/) — compose_vars 拉 mysql 凭据 + init_files 创建配置文件
5. **自定义镜像应用**：[`php_80/`](../php_80/) — Dockerfile + GitHub Actions 自动构建推送 Docker Hub

---

## 文档维护

本文档版本：v1.3（2026-06-07）

修改历史：
- v1.3 — 新增常见错误 #10~#15（PHP fpm_port 移除、manage 必须有 file_path、MySQL innodb_log_file_size 移除、compose 挂载文件必须有 init_files、default_version 不应用 rolling tag、EOL 版本标 deprecated）；更新附录 B 移除 wordpress 参考（已从 apps-repo 删除）
- v1.2 — 新增网络模式规范（host vs panel_network 完整决策树）；新增常见错误 #9（PHP localhost 连不上 MySQL）；更正 docker-compose 模板规范中"所有应用必须挂 panel_network"的错误表述
- v1.1 — 新增 `bool_format` 字段说明（manage → type: switch 必填）；新增 `format` 字段说明；新增常见错误 #8
- v1.0 — 初版，覆盖所有 yaml 字段、compose 模板、占位符、CI 校验、新应用流程

如发现文档与代码不一致，**以代码（backend/internal/service/app_types.go）为准**，并提 PR 修正本文档。
