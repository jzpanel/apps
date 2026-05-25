# jzpanel/apps

极致面板（jzpanel）应用商店。每个应用都是一个目录，包含 `app.yaml` + `docker-compose.yml` + 图标。

面板会从 jsDelivr CDN 拉取 `version.json` 检测更新，从 codeload.github.com 下载本仓库 tarball 解压到 `/www/server/panel/apps/`。

## 仓库结构

```
apps-repo/
├── docs/
│   └── APP_AUTHORING_GUIDE.md       # 应用编写规范（写新应用必读）
├── scripts/
│   ├── check-versions.go             # CI：校验所有 tag manifest 真实存在
│   ├── lint-volumes-init.go          # CI：检查非 root 镜像必须声明 volumes_init
│   └── compute-bump.go               # CI：版本号自动 bump
├── .github/workflows/
│   ├── validate.yml                  # PR/push 触发 CI 校验
│   ├── cron-validate.yml             # 每天定时跑 strict 校验
│   └── bump.yml                      # yaml 改动后自动 bump version.json
├── version.json                      # 应用商店版本号（CI 自动维护）
├── renovate.json                     # Renovate 自动 PR 升级配置
├── <app-key>/                        # 各个应用目录
│   ├── app.yaml
│   ├── docker-compose.yml
│   ├── icon.svg
│   └── init/                         # 初始化文件（按需）
└── README.md                         # 本文件
```

## 添加新应用

请阅读 **[docs/APP_AUTHORING_GUIDE.md](docs/APP_AUTHORING_GUIDE.md)**，文档完整覆盖：
- app.yaml 所有字段及含义
- docker-compose.yml 模板规范
- 模板占位符列表
- 挂载目录权限（volumes_init）
- 初始化文件、动态变量、服务依赖
- 资源限制、反代网站
- CI 校验逻辑
- 新增应用的完整流程
- 常见错误与排查

## 修改已有应用

- 修改 yaml/compose 字段：直接修改即可
- 升级镜像版本：改 `versions[].tag`，CI 会校验真实存在性
- yaml 改动 push 后自动触发 `bump.yml` 升 patch 版本号

## 本地校验

```bash
cd scripts
go run check-versions.go --apps-dir ..
go run lint-volumes-init.go --apps-dir ..
```

## CI 流程

每个 PR 会跑：
1. **validate.yml** — `check-versions.go` 校验所有 tag 的 manifest 在 Docker Hub 真实存在
2. **validate.yml** — `lint-volumes-init.go` 校验非 root 镜像必须声明 volumes_init

每天 UTC 凌晨：
- **cron-validate.yml** — 严格模式（含 deprecated 版本）校验，捕获上游主动删 tag

push 到 main 后：
- **bump.yml** — 自动 bump version.json 的 patch 号，调用 jsDelivr purge

## 许可

应用商店元数据采用 MIT 许可。各应用本身的镜像许可遵循上游声明（postgres/mysql 等遵循各自许可）。
