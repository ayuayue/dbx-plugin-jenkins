# Jenkins Console · DBX 插件

在 DBX 内管理 Jenkins 任务、构建、控制台日志、队列与节点。

- 插件 ID：`io.github.nixing178.jenkins`
- 当前版本：`0.4.0`（阶段视图支持悬停查看变更历史）
- 形态：Go Sidecar + Vue3 静态工作台
- 许可证：Apache-2.0

## 目录结构

```
manifest.json              插件身份、连接表单、贡献点
dbx-plugin.toml            打包配置（后端 + include）
index.html                 Vite 入口
src/                       Vue3 工作台源码
ui/                        构建产物（pnpm build 生成，打包时使用）
backend/
  main.go                  Sidecar 入口（JSON-RPC）
  internal/jenkins/        Jenkins 客户端与纯函数（TDD 覆盖）
  third_party/             内置 Go SDK（离线构建用）
assets/                    插件与连接图标
dist/                      .dbxp 打包产物
```

## 环境要求

| 工具 | 版本 |
| --- | --- |
| Node.js | >= 22 |
| pnpm | 10.x |
| Go | 1.22 |
| DBX | >= 0.6.43（本机 0.6.47） |

## 安装依赖

```powershell
cd E:\DBX插件开发\Jenkins
pnpm install
```

> pnpm 10 默认不执行依赖的 postinstall。若遇到 esbuild 的 `EBUSY` 校验失败，使用 `pnpm install --ignore-scripts`；esbuild 的 Windows 二进制来自 `@esbuild/win32-x64`，不受影响。

## 构建 UI

```powershell
pnpm build        # vue-tsc 类型检查 + vite 单文件构建，输出到 ui/
```

## 测试（TDD）

> 测试按 TDD 红→绿推进，历史红/绿证据见 [`docs/test-evidence.md`](./docs/test-evidence.md)。

后端全量：

```powershell
$env:PATH = "<Go 安装目录>\bin;" + $env:PATH
cd E:\DBX插件开发\Jenkins\backend
go test ./... -v
```

前端（Vitest）：

```powershell
cd E:\DBX插件开发\Jenkins
pnpm test
```

## 真实 Jenkins E2E

已在临时 Jenkins **2.586**（Java 21，`/jenkins` 前缀，CSRF 开启）跑通：

- 连接 / whoami / 任务 / 参数 / 构建 / 控制台 / 停止 / 队列 / 节点 / 概览
- 任务按 Jenkins 原生 Views 分组（all/Builds/Ops 已实测）
- Pipeline 任务支持阶段视图：构建 × stage 矩阵、平均耗时、失败/进行中/跳过着色；行首提交数悬停/点击展开变更浮层（哈希、信息、作者），可跳转构建详情
- 无参与带参构建、只读拦截、MCP `list_jobs`
- 完整步骤与实测输出：[`docs/e2e-jenkins.md`](./docs/e2e-jenkins.md)
- 脚本：[`scripts/e2e-jenkins.js`](./scripts/e2e-jenkins.js)

## 视图 / 任务管理

连接处于**可写模式**（关闭「只读模式」）时，可以在 DBX 内直接管理 Jenkins 配置：

| 操作 | 入口 | 说明 |
| --- | --- | --- |
| 新建任务 | 任务列表顶部 `＋` | 自由风格 / Pipeline；在左侧当前文件夹内创建 |
| 编辑任务 | 任务详情右上「编辑」 | 表单改名称、描述、启用状态、Pipeline 脚本；高级页签可直接改 config.xml |
| 删除任务 | 任务详情右上「删除」 | 需手动输入任务名确认；同时删除全部构建历史 |
| 新建视图 | 面包屑右侧 `＋` | ListView；可勾选任务、填包含正则、递归子文件夹 |
| 编辑视图 | 面包屑右侧 `⋯` | 表单改名称、描述、任务清单、包含正则；非 ListView 仅支持 XML |
| 删除视图 | 面包屑右侧 `⋯` | 需手动输入视图名确认；不会删除视图内任务 |

实现要点：

- 编辑配置采用 **XML 局部改写**，只替换 `description` / `disabled` / `script` / `jobNames` 等目标节点，SCM、构建步骤、插件字段原样保留。
- **视图改名**走 `POST /view/<name>/configSubmit`（`config.xml` 提交路径明确拒绝覆盖名称），并提交完整结构化表单，避免 Jenkins 全量重建时清空任务与列；改名后再用原 XML 回写一次，恢复 `properties` / `jobFilters` 等表单覆盖不到的字段。
- Pipeline 脚本与 config.xml 使用内嵌代码编辑器：Groovy / XML 语法高亮、行号、当前行高亮，`Tab` / `Shift+Tab` 缩进，右上「展开」切全屏编辑（`Esc` 先收起编辑器，再按才关闭窗口）。零外部依赖，产物只增约 1 KB。
- 表单只提交**发生变化的字段**，未修改项不会被重写。
- 所有写操作复用后端的只读守卫、CSRF crumb 刷新重试与权限错误映射。
- **新建任务后列表自动刷新**：在自定义 ListView 下新建任务，会把任务追加进该视图的 `jobNames`（只追加、以服务端最新清单为基线，不动描述 / 列 / 过滤规则），随后刷新列表并选中；正则视图已匹配时不产生写请求；非 ListView（如聚合视图）只提示「已创建，但未能加入视图」。
- Jenkins 返回 HTML 错误页时，提示会收敛为「HTML error page (HTTP xxx)」，不再把整页 HTML 塞给用户。

所需 Jenkins 权限：`Item/Create`、`Item/Configure`、`Item/Delete`、`View/Create`、`View/Configure`、`View/Delete`。

## 自动刷新

- **构建列表**：当前任务的构建里存在「构建中」时，每 3s 静默刷新任务详情与最近 20 次构建；最后一条构建结束后自动停止。
- **刚触发的构建**：点「带参数构建」/「立即构建」后立即静默补刷一次、800ms 再补一次，并持续跟踪到新构建出现在列表里（最长 120s，排队过久自动放弃）。
- **阶段视图**：存在构建中或 `IN_PROGRESS` 的 stage 时，每 5s 静默刷新最近 10 次构建的阶段矩阵。
- **左侧任务列表**：触发构建后立即静默刷新一次、800ms 再补一次；新构建出现时刷一次（圆点变「构建中」、`#构建号` 更新），构建结束、轮询停止时再补一次收敛。构建期间不额外轮询列表，避免无收益的重复请求。
- 静默刷新不会闪「加载中…」占位，请求失败也不弹错误横幅；手动刷新会重置计时，避免刚点完又自动刷一次。
- 切换任务 / 视图、进入构建详情、关闭工作台即停止；页面不可见时跳过请求，回到前台后的下一轮自动恢复。
- 面板标题旁的绿点「自动刷新中」表示轮询正在进行。

## 打包

```powershell
$env:PATH = "<Go 安装目录>\bin;" + $env:PATH
cd E:\DBX插件开发\Jenkins
node scripts/check-version.mjs          # manifest / package.json / main.go / UA 版本必须一致
pnpm build
npx --yes @dbx-app/plugin-cli@0.1.9 package .
node scripts/verify-sidecar-identity.mjs dist\dbx-plugin-jenkins.exe   # 校验后端 identity
```

> **版本号必须同步**：DBX 会比对 `manifest.version` 与 sidecar `plugin/initialize` 返回的版本，
> 不一致会直接拒绝启动（`backend identity ... does not match manifest`）。
> 版本号分散在 `manifest.json`、`package.json`、`backend/main.go`、`backend/internal/jenkins/client.go` 的 User-Agent 四处，
> 由 `check-version.mjs` 与 `verify-sidecar-identity.mjs` 双重把关。

产物：

```
dist/io.github.nixing178.jenkins-0.4.0-windows-x64.dbxp
dist/io.github.nixing178.jenkins-0.4.0-windows-x64.artifact.json
```

## 发布到 DBX 插件商店

商店目录在 [`t8y2/dbx-store`](https://github.com/t8y2/dbx-store)，发布分两步：作者发候选包，维护者审核签名。

**1. 发候选 Release（CI 自动构建）**

`ui/` 与 `dist/` 不进 Git，候选包由 CI 从源码重建，因此本地 `dist/` 的哈希不能用于上架。

```bash
git tag v0.4.0
git push origin main --tags
# GitHub → Releases → Draft a new release → 选 v0.4.0 → Publish
```

`.github/workflows/plugin-release.yml` 会复用官方 `plugin-release-reusable.yml`，自动产出并上传三个资产：

| 资产 | 说明 |
| --- | --- |
| `io.github.nixing178.jenkins-<version>-windows-x64.dbxp` | 未签名候选包 |
| `io.github.nixing178.jenkins-<version>-windows-x64.artifact.json` | 目标平台、sha256、size |
| `release-candidates.json` | 汇总，供商店同步器读取 |

**2. 提 catalog PR**

向 `t8y2/dbx-store:main` 提交单个 PR，只加两个文件：

- `publishers/nixing178.json`（首次上架）
- `candidates/io.github.nixing178.jenkins.json`（`sha256`/`size` 必须取自 CI 产出的 `.artifact.json`）

维护者评论 `/sign` 后，签名工作流会回写 `plugins/io.github.nixing178.jenkins.json` 与 `catalog/index.json`，合并即上架。
签名私钥由商店维护，作者不接触。

> 已签名版本不可覆盖：任何改动都必须升 semver，并同步四处版本源（见上节）。

## 本地安装

1. DBX → 插件中心 → 设置 → 打开「允许安装未签名开发包」。
2. 插件中心 → 安装 → 从文件 → 选择 `dist/io.github.nixing178.jenkins-0.4.0-windows-x64.dbxp`。
3. 新建连接，类型选 **Jenkins**，填写地址、端口、用户名与 API Token。
4. 从 `com.anjun.jenkins` 旧版升级时：两者是不同插件，需先在插件中心卸载旧版，再安装新版。

> 若启动时报 `Plugin 'io.github.nixing178.jenkins' initialization failed: backend identity 'x' does not match manifest 'y'`，
> 说明包内后端版本与 manifest 不一致（0.2.0 曾有此问题）。改用 0.3.0+ 的包，并先卸载旧版。
> 插件 ID 于 0.3.0 从 `com.anjun.jenkins` 迁移到 `io.github.nixing178.jenkins`，DBX 视为两个独立插件，
> 旧版的 Jenkins 连接不会自动迁移，需在新插件下重新创建。

## 连接配置

| 字段 | 说明 |
| --- | --- |
| 协议 | HTTPS（默认）/ HTTP（内网） |
| IP / 域名 + 端口 | Jenkins 控制器地址，默认 `8080` |
| 上下文路径 | 部署在子路径时填写，如 `/jenkins` |
| 用户名 + API Token | 在 Jenkins「用户设置 → API Token」生成；留空表示匿名只读 |
| 只读模式 | 默认开启，后端会拒绝触发/停止/取消队列 |
| 跳过 TLS 证书校验 | 仅用于自签证书的内网实例 |

## AI / MCP 工具

在「插件中心 → 已安装 → 内置 AI 工具」开启后，AI 可使用：

| 工具 | 读写 | 说明 |
| --- | --- | --- |
| `list_jobs` | 只读 | 列出根目录/文件夹任务，支持名称过滤 |
| `get_job` | 只读 | 任务详情与参数定义 |
| `list_builds` | 只读 | 最近构建列表 |
| `get_build` | 只读 | 单次构建详情、变更集、产物 |
| `get_console_log` | 只读 | 增量控制台日志（用 `nextStart` 续读） |
| `list_queue` | 只读 | 队列与阻塞原因 |
| `list_nodes` | 只读 | 节点状态与执行器 |
| `trigger_build` | 写 | 触发构建（可有参），宿主逐次确认 |
| `stop_build` | 写 | 停止构建（destructive） |
| `cancel_queue_item` | 写 | 取消排队（destructive） |

只读连接下，3 个写工具不会出现在工具列表中；后端仍会二次拦截。

## 当前进度

| 里程碑 | 状态 |
| --- | --- |
| M0 脚手架 / manifest / 打包链路 | 已完成 |
| M1 连接、认证、CSRF、错误映射 | 已完成（75 后端用例 + 6 前端用例全绿） |
| M2 任务、构建、控制台 | 已完成（102 后端用例 + 18 前端用例全绿，含 sidecar 冒烟） |
| M3 队列、节点、概览 | 已完成（111 后端用例全绿，M3 端点冒烟通过） |
| M4 MCP 工具面 | 已完成（120 后端用例全绿，10 工具冒烟通过） |
| M5 打包验收 | E2E 已完成（Jenkins 2.586 全链路通过）；待本机安装验收 |
| M6 视图 / 任务 CRUD | 已完成（84 jenkins + 45 rpc 用例；真实 Jenkins 2.586 建/改/删/重命名/只读拦截 E2E 通过） |
| M7 运行时自动刷新 | 已完成（构建列表 3s / 阶段视图 5s 静默轮询，构建结束自动停止） |
| M8 触发后跟踪新构建 | 已完成（触发即补刷 + 持续跟踪至新构建出现，120s 上限；114 前端用例全绿） |

> 任务列表已按 Jenkins 原生 Views 分组：顶部 Tab 切换视图，视图内分「文件夹 / 任务」两组。
> Tab 溢出时左右箭头滚动，当前视图自动居中；打开视图自动选中首个任务；左栏宽 320px。
> 当前测试基线：129 后端用例 + 102 前端用例。
>
> Pipeline 创建依赖 Jenkins 的 `workflow-job` / `workflow-cps` 插件；实例未安装时 Jenkins 返回 500，插件会给出去敏的 HTML 错误提示，不会创建半成品任务。

![Jenkins Views 分组](docs/screenshots/jenkins-views.png)
![Stage View](docs/screenshots/stage-view.png)
