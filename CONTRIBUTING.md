# 贡献指南

感谢你想为 jzpanel/apps 贡献新应用或修复问题。

## 必读文档

**所有贡献者必须先阅读：[docs/APP_AUTHORING_GUIDE.md](docs/APP_AUTHORING_GUIDE.md)**

文档涵盖了 yaml schema 所有字段、compose 模板规范、占位符、权限处理、CI 流程、新应用编写步骤等。不读文档直接提 PR 大概率不会通过 CI。

## 快速 Checklist

提 PR 前自查：

### 必要项

- [ ] 应用目录命名是 `<key>`，全小写下划线分隔
- [ ] 含 `app.yaml`、`docker-compose.yml`、`icon.svg`
- [ ] `app.yaml` 含 `key/name/category/description/versions/default_version/icon`
- [ ] `versions[].tag` 是精确 tag（不是 `latest`、不是 `1.27` 这种 floating tag）
- [ ] `versions[].image` 与 `docker-compose.yml` 里 image 字段一致
- [ ] `versions[].architectures` 声明（推荐 `[amd64, arm64]`）
- [ ] `docker-compose.yml` 用 `{{tag}}` 占位符，`container_name: panel_<key>`
- [ ] 加入 `panel_network`（external: true）

### 非 root 镜像（postgres/mysql/mongo/redis 等）

- [ ] 声明 `volumes_init` 含数据目录和日志目录的 uid/gid
- [ ] CI 的 `lint-volumes-init.go` 必须通过

### 端口

- [ ] 所有外部端口用 `{{port}}` / `{{http_port}}` 等 param 占位符
- [ ] 默认端口避开常见冲突（80/443/3306/5432 这些已有应用占用）

### 配置文件

- [ ] 配置文件放 `init/` 目录下
- [ ] `init_files` 中 `path` 与 `docker-compose.yml` 挂载源一致
- [ ] 配置文件里的可调参数用 `{{param_key}}` 占位

### 测试

- [ ] 在 dev 服务器上手动安装该应用，容器健康
- [ ] 卸载能完整清理（容器、数据目录、配置文件）
- [ ] 至少一种支持的架构（amd64）实测过

### CI

- [ ] 本地跑过 `go run scripts/check-versions.go --apps-dir ..`
- [ ] 本地跑过 `go run scripts/lint-volumes-init.go --apps-dir ..`

## PR 描述模板

```markdown
## 添加 <Name> 应用

### 应用简介
（一两句说明该应用是什么）

### 测试过的版本
- display: "8.0", tag: "8.0-alpine" — ✓ amd64 实测

### 容器内进程 UID
（如果是非 root 镜像，列出 `docker run --rm <image> id` 输出）

### 端口
- HTTP: 8080
- ...

### 备注
（其他需要 reviewer 知道的信息）
```

## 修改已有应用

- **升级镜像版本**：直接改 `versions[].tag`，CI 会校验
- **加新版本**：在 `versions[]` 数组追加，记得标 `is_default` 或更新 `default_version`
- **废弃老版本**：标 `deprecated: true`，半年后删除
- **改 yaml 字段**：参考 [APP_AUTHORING_GUIDE.md](docs/APP_AUTHORING_GUIDE.md) 了解字段语义

## 沟通

- 不确定的设计选择 → 先开 issue 讨论
- 紧急安全问题 → 直接联系维护者，不要走 public PR

## 许可

提 PR 即表示你接受将贡献以 MIT 许可发布。
