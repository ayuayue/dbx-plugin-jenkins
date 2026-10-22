# 测试证据

> 后端 `go test` 与前端 `pnpm test` 按 TDD 红→绿推进，每轮结果记录如下。

## 环境

| 项 | 值 |
| --- | --- |
| Go | 1.22.12 |
| Node | 22.22.2 |
| Vitest | 2.1.9 |
| 执行时间 | 2026-10-19 |

## 汇总

| 轮次 | 范围 | 命令 | 红 | 绿 |
| --- | --- | --- | --- | --- |
| M0-B1 | 路径构造 + 状态归一化 | `go test ./internal/jenkins/ -v` | 32 FAIL / 41 | 41 PASS |
| M1-C1 | Client（Auth / CSRF / 只读 / 错误映射 / TLS） | `go test ./internal/jenkins/ -v` | 12 FAIL / 16 | 61 PASS |
| M1-R1 | RPC 生命周期 + secret 形态 + 会话 | `go test ./internal/rpc/ -v` | 14 FAIL / 14 | 75 PASS（全量） |
| M1-F1 | 前端错误映射 | `pnpm test` | 模块缺失，1 suite FAIL | 6 PASS |
| M2-B2 | 任务 / 构建 / 控制台 / 触发停止 | `go test ./internal/jenkins/ -v` | 16 FAIL / 21 | 82 PASS |
| M2-R2 | RPC jobs/job/builds/build/console/trigger/stop | `go test ./internal/rpc/ -v` | 6 FAIL | 102 PASS（全量） |
| M2-F2 | 前端状态/时长/日志纯函数 | `pnpm test` | 3 suite 模块缺失 | 18 PASS |
| M2-S1 | Sidecar 二进制 JSON-RPC 冒烟 | `node sidecar-smoke.js` | — | init/connect/jobs/disconnect OK |
| M3-B3 | 队列 / 节点 / 概览聚合 | `go test ./internal/jenkins/ -v` | 5 FAIL | 111 PASS（全量） |
| M3-R3 | RPC overview/queue/cancelQueue/nodes | `go test ./internal/rpc/ -v` | 4 FAIL | 111 PASS（全量） |
| M3-S2 | Sidecar 冒烟（M3 端点） | `node sidecar-smoke.js` | — | overview/queue/nodes/cancelQueue OK |
| M4-B4 | MCP tools/call（10 工具、只读过滤、注解） | `go test ./internal/rpc/ -v` | 9 FAIL | 120 PASS（全量） |
| M4-S3 | Sidecar 冒烟（MCP） | `node sidecar-smoke.js` | — | 10 tools 列出 + list_jobs 调用成功 |
| M5-E2E | 真实 Jenkins 2.586 全链路 | `node scripts/e2e-jenkins.js` | — | 连接/任务/构建/参数/控制台/停止/队列/节点/概览/MCP/只读全部 OK |
| M5-REG | `computer.idle` 布尔值回归 | `go test ./internal/jenkins/ -run TestGetNodes` | — | 121 PASS（全量） |
| V1 | Jenkins Views API + 视图任务 | `go test ./internal/jenkins/ -v` | 4 FAIL | 130 PASS（全量） |
| V2 | RPC `jenkins/views`、jobs 按视图、MCP view | `go test ./internal/rpc/ -v` | 4 FAIL | 130 PASS（全量） |
| V3 | 前端 `groupJobs` | `pnpm test` | 模块缺失 | 21 PASS |
| V4 | 真实 Jenkins Views E2E | `node scripts/e2e-jenkins.js` | — | all/Builds/Ops 分组正确 |
| UI1 | 视图 Tab 溢出 + 首屏自动选中 | Chrome headless 截图 | — | 5 视图/22 任务：当前 Tab 完整居中、箭头可见、右栏自动加载首个任务 |
| UI2 | 视图 Tab 鼠标滚轮横向滚动 | `pnpm test` + Chrome CDP 真实滚轮 | 1 suite FAIL（`./tabs` 模块缺失） | 38 PASS；滚轮 0→150→550→586（钳制在 max）、边界外保持 586、反向回滚 436、页面 scrollTop 恒为 0 |
| ST1 | Stage `wfapi/describe` + 矩阵聚合 | `go test ./internal/jenkins/ -v` | 5 FAIL | 137 PASS（全量） |
| ST2 | RPC `jenkins/stages` | `go test ./internal/rpc/ -v` | 2 FAIL | 137 PASS（全量） |
| ST3 | 前端阶段耗时/色调/强度 | `pnpm test` | 模块缺失 | 28 PASS |
| ST4 | 真实 Pipeline 阶段视图 E2E | `node scripts/e2e-jenkins.js` | — | 7 stages / 5 builds / WorkflowJob 识别通过 |
| CRUD-B1 | 名称校验 / config.xml 路径 / 创建·编辑·删除任务 / 创建·编辑·删除视图 / 只读守卫 | `go test ./internal/jenkins/ -v` | 27 FAIL / 45 PASS | 84 PASS |
| CRUD-B2 | XML 保守改写（description/disabled/script/ListView） | `go test ./internal/jenkins/ -run TestPatch -v` | 6 FAIL | 11 PASS |
| CRUD-B3 | 增量更新（改名→读原文→局部改写→写回） | `go test ./internal/jenkins/ -run TestUpdate -v` | — | 9 PASS |
| CRUD-R1 | RPC 9 个 CRUD 方法 + 只读拦截 | `go test ./internal/rpc/ -v` | 10 FAIL | 45 PASS |
| CRUD-F1 | 前端名称校验/载荷/差异字段/删除确认 | `pnpm test` | 1 suite FAIL（`./forms` 模块缺失） | 55 PASS |
| CRUD-E1 | 真实 Jenkins 2.586 CRUD E2E | `node scripts/e2e-jenkins.js` | — | 视图建/改/删 + 任务建/改/禁用/重命名/删 + 只读 3 项拦截全部通过；Pipeline 创建因实例缺 workflow 插件跳过 |
| CRUD-E2 | HTML 错误页摘要 | `go test ./internal/jenkins/ -run TestAPIError` | — | 2 PASS |
| UI3 | Pipeline / XML 编辑器：语法高亮 + 行号 + 当前行 + 展开全屏 + Tab 缩进 | `pnpm test` + Chrome CDP 像素校验 | 2 suite FAIL（`./highlight`、`./editor` 模块缺失） | 82 PASS（全量）；行高 18.6 / 内距 10 两侧一致、高亮层与 textarea `scrollHeight` 均为 392、展开 900×760 突破 modal `overflow:hidden`、Esc 先收编辑器再关弹窗、Tab 首行 `pipeline` → `  pipeline`（caret 2） |
| AR1 | 自动刷新判定 + 串行轮询器 | `npx vitest run src/lib/refresh.test.ts` | 1 suite FAIL（`./refresh` 模块缺失） | 20 PASS；全量 102 PASS（11 files） |
| CH1 | 阶段矩阵变更集：`changeSet`/`changeSets` 合并去重、200 rune 截断、空数组序列化、`jenkins/builds` 契约不变 | `go test ./internal/jenkins/ -run "TestGetStageMatrix\|TestGetBuild"` | build failed（10 处未定义字段/常量） | 15 PASS；全量 141 PASS |
| CH1b | Pipeline `changeSets`（复数）真机缺陷回归 | 同上 | 5 FAIL（tree 缺 `changeSets`、`commitCount=0`） | 15 PASS |
| CH2 | 前端浮层纯函数：短哈希 / 首行摘要 / 视口定位与翻转 | `pnpm test` | 11 FAIL / 114 PASS（函数未定义） | 125 PASS（11 files） |
| CH3 | 浮层视觉与交互：Teleport 不被裁切、贴边翻转、8 条滚动、hover/键盘/Esc/外部点击/详情跳转 | Chrome CDP + headless 截图 | — | `parent=BODY`、`clippedBy=null`、8 条 `listScrollable=true`、底部行翻转 `top=92`、点击详情 `openBuild=38`、Esc/外部点击均关闭 |
| CH4 | 真实 Jenkins 2.586 Pipeline changelog：`jenkins/stages` 返回 `changes` | 临时 sidecar 探针（已删） | 修复前 `commitCount=0`、`changes=[]` | `commitCount=1`、`changes=[945f1594 / feat(export): 阶段视图变更浮层联调提交 / e2e]` |

## 关键红测样本

### M0：未实现桩

```
--- FAIL: TestJobPath/nested_folder
    paths_test.go:...: JobPath() = "", want "/job/team/job/api/job/build"
--- FAIL: TestNormalizeColor/red
    status_test.go:31: NormalizeColor("red") = "unknown", want "failure"
```

### M1：Client 未实现

```
--- FAIL: TestPostFormRetriesWithFreshCrumb
    client_test.go:172: NewClient() error = not implemented
--- FAIL: TestReadOnlyBlocksWrites
    client_test.go:223: NewClient() error = not implemented
```

### M1：RPC 未实现

```
--- FAIL: TestConnectionTestTokenShapes/secrets
    service_test.go:...: connection/test error = not implemented
--- FAIL: TestConnectDisconnectIdempotent
    service_test.go:215: connect #1 error = not implemented
```

### M1：前端模块缺失

```
FAIL  src/lib/errors.test.ts
Error: Failed to load url ./errors
```

### M2：任务/构建/控制台未实现

```
--- FAIL: TestGetJobsRoot
    jobs_test.go:30: GetJobs() error = not implemented
--- FAIL: TestGetConsoleProgressive
    console_test.go:39: GetConsole(first) error = not implemented
--- FAIL: TestTriggerBuildWithParams
    actions_test.go:...: TriggerBuild() error = not implemented
```

### M2：RPC 方法不存在

```
--- FAIL: TestRPCJobsAndJob
    m2_test.go:108: jobs error = method not found: jenkins/jobs
--- FAIL: TestRPCTriggerReadOnly
    m2_test.go:263: error = method not found: jenkins/trigger, want read_only
```

### M2：前端纯函数模块缺失

```
FAIL  src/lib/status.test.ts
FAIL  src/lib/format.test.ts
FAIL  src/lib/console.test.ts
```

### M3：队列/节点/概览未实现

```
--- FAIL: TestGetQueue
    system_test.go:28: GetQueue() error = not implemented
--- FAIL: TestGetOverview
    system_test.go:145: GetOverview() error = not implemented
--- FAIL: TestRPCOverview
    m3_test.go:64: overview error = method not found: jenkins/overview
```

### M4：MCP 方法未实现

```
--- FAIL: TestMCPToolsReadOnly
    mcp_test.go:...: mcp/tools error = method not found
--- FAIL: TestMCPCallTriggerReadOnly
    mcp_test.go:241: mcp/call error = method not found: mcp/call
```

### V1/V2：Views 未实现

```
--- FAIL: TestGetViews
    views_test.go:30: GetViews() error = not implemented
--- FAIL: TestRPCViews
    views_rpc_test.go:68: jenkins/views error = method not found
--- FAIL: TestMCPListJobsSchemaHasView
    views_rpc_test.go:180: list_jobs schema missing view
```

### ST1/ST2：阶段视图未实现

```
--- FAIL: TestGetBuildStages
    stages_test.go:30: GetBuildStages() error = not implemented
--- FAIL: TestGetStageMatrix
    stages_test.go:97: GetStageMatrix() error = not implemented
--- FAIL: TestRPCStages
    stages_rpc_test.go:54: stages error = method not found: jenkins/stages
```

### UI2：滚轮换算模块缺失

```
FAIL  src/lib/tabs.test.ts [ src/lib/tabs.test.ts ]
Error: Failed to load url ./tabs (resolved id: ./tabs) in .../src/lib/tabs.test.ts
Test Files  1 failed (1)   Tests  no tests
```

修复后：`src/lib/tabs.test.ts (10 tests)` 通过，全量 `38 passed (38)`。

## 视图 Tab 滚轮 UI 验证（Chrome 147 headless + CDP，2026-10-19）

真实 `Input.dispatchMouseEvent(type=mouseWheel)` 事件打到 `.view-tabs` 上（14 个视图，maxScroll=586）：

| 步骤 | scrollLeft |
| --- | --- |
| 初始 | 0 |
| 向下滚 150 | 150 |
| 再向下滚 400 | 550 |
| 滚到底（deltaY=4000） | 586（= maxScroll，不越界） |
| 边界外继续向下 | 586 |
| 向上滚 150 | 436 |
| 页面 scrollTop | 0（未把外层页面带着滚，`preventDefault` 生效） |

截图：`docs/screenshots/view-tabs-wheel.png`

## Sidecar 冒烟（真实二进制）
用 mock Jenkins（`127.0.0.1:18080`）驱动编译后的 `dbx-plugin-jenkins.exe`，按 DBX 协议发送 JSON-RPC：

```
init: {"capabilities":["connections"],"plugin":{"id":"com.anjun.jenkins","version":"0.1.0"},"protocolVersion":1}
connect: {"message":"Connected to Jenkins 2.440.3","readOnly":false,"success":true,"user":{"name":"u","authenticated":true,...}}
jobs: {"jobs":[{"name":"demo","fullName":"demo","status":"success","isFolder":false,...}]}
mcp/tools: 10 list_jobs,get_job,list_builds,get_build,get_console_log,list_queue,list_nodes,trigger_build,stop_build,cancel_queue_item
mcp/call list_jobs: {"content":[{"type":"text","text":"{\"jobs\":[...]}"}],"isError":false}
overview: {"version":"2.440.3","user":"u","mode":"NORMAL","queueLength":1,"nodesOnline":1,"nodesTotal":1,"executorsBusy":2,"executorsTotal":4}
queue: {"items":[{"id":42,"jobName":"team/demo","why":"Waiting for next available executor",...}]}
nodes: {"nodes":[{"name":"Built-In Node","offline":false,"numExecutors":4,"busy":2,"labels":["built-in"]}]}
cancelQueue: {"success":true}
disconnect: {"success":true}
```

## 真实 Jenkins E2E（2026-10-19）

Jenkins 2.586 + Java 21，临时 JENKINS_HOME，`/jenkins` 前缀，CSRF 开启：

```text
connect: Connected to Jenkins 2.586
jobs: demo, param-demo, pipeline-demo, slow-job
views: Builds, Ops, all
Builds view: demo, param-demo
Ops view: slow-job
param-demo parameters: BRANCH=main
demo build: SUCCESS, console contains "hello from dbx"
param-demo: console contains "branch=dev"
pipeline class: org.jenkinsci.plugins.workflow.job.WorkflowJob
stages: 获取构建人信息,参数验证,代码检出,打包构建,备份回滚应用,部署,启动应用 | builds 5 | avgRun 20961
stage build: {"number":5,"commits":0,"stages":7}
slow-job: ABORTED after stop
nodes: Built-In Node 0/2 busy
overview: executors 2, nodes 1/1
mcp tools: 10
read-only trigger: read_only
E2E OK
```

### E2E 发现并修复

`computer.idle` 在真实 Jenkins 里是**布尔值**，原实现按 int 解码导致 `jenkins/nodes` 报
`cannot decode Jenkins response`。修复为 `executors[idle]` 明细统计空闲数，旧版本无明细时退化为布尔值；
新增 `TestGetNodesFallbackIdleFlag` 回归测试。

## 绿测结果

```
ok  github.com/anjun/jenkins/internal/jenkins
ok  github.com/anjun/jenkins/internal/rpc

Test Files  6 passed (6)
Tests       28 passed (28)
```

> Vitest 配置 `fileParallelism: false`：宿主 FS shim 偶发拦截并行 worker 的临时 SSR 文件（EPERM），串行执行后稳定。

### CRUD：桩实现未落地

```
--- FAIL: TestCreateView
    config_test.go:...: CreateView() error = nil, want a request
--- FAIL: TestPatchDescriptionReplacesAndPreserves
    xmlpatch_test.go:64: description not replaced/escaped:
--- FAIL: TestRPCUpdateJob
    config_rpc_test.go:...: updateJob error = &{Code:-32601 Kind:method_not_found}
```

### CRUD：XML 1.1 声明

```
--- FAIL: TestPatchDisabled
    xmlpatch_test.go:99: PatchDisabled() error = invalid XML: xml: unsupported version "1.1"; only version 1.0 is supported
```

修复：扫描前把声明里的 `1.1` 等长替换成 `1.0`，原文与偏移保持不变。

### AR：自动刷新模块缺失

```
FAIL  src/lib/refresh.test.ts [ src/lib/refresh.test.ts ]
Error: Failed to load url ./refresh (resolved id: ./refresh) in E:/DBX插件开发/Jenkins/src/lib/refresh.test.ts. Does the file exist?
 Test Files  1 failed | 10 passed (11)
      Tests  82 passed (82)
```

实现 `src/lib/refresh.ts` 后 20 PASS，全量 102 PASS（11 files）。覆盖：

- `hasRunningBuild`：空数组 / `null` / 全部结束 / `building: true` / `status: "building"`；
- `hasRunningStage`：空矩阵 / 全部结束 / 构建 `building` / stage `IN_PROGRESS`、`in_progress`、`BUILDING` / `stages: null`；
- `createAutoRefresher`：未到间隔不执行、到点执行并续排、上一轮未完成不重入、`stop()` 终止、重复 `start()` 幂等、`stop()` 后可重新 `start()`、`shouldContinue` 返回 false 自动停、任务抛错后继续、`restart()` 重置计时、未启动时 `restart()` 为空操作；
- 常量：构建列表 3000ms、阶段视图 5000ms。

### AS：触发构建后不主动刷新

红（只加了测试，未实现）：

```
 ❯ src/lib/refresh.test.ts (0 test)
 FAIL  src/lib/refresh.test.ts [ src/lib/refresh.test.ts ]
TypeError: createPendingBuild is not a function
 ❯ src/lib/refresh.test.ts:320:19
 Test Files  1 failed | 10 passed (11)
      Tests  82 passed (82)
```

绿（实现 `hasNewBuild` / `createPendingBuild` / `evaluatePendingBuild` + `App.vue` 接线）：

```
 ✓ src/lib/refresh.test.ts (32 tests)
 Test Files  11 passed (11)
      Tests  114 passed (114)
```

`pnpm build`：`[check-version] ok: 0.3.0` → `vue-tsc --noEmit` 无错误 → `ui/index.html 191.48 kB`。

新增覆盖：

- `hasNewBuild`：空 / `null` / `undefined`；只有触发前的编号 → false；出现更大编号 → true；首次构建（`previousNumber = 0`）→ true；上一个构建仍在 `building` 但编号未涨 → false（避免误判提前停止跟踪）；
- `createPendingBuild`：绑定任务名、触发前编号与 `deadline = now + 120000`；省略 `now` 时以当前时间为准；
- `evaluatePendingBuild`：任务切换 → `stale`；出现新构建 → `settled`；`settled` 优先于 `expired`；超时无新构建 → `expired`；未到期 → `watching`；
- 接线（`App.vue`）：`pendingBuild` 并入 `autoRefreshMode` 的 `builds` 分支（不新建定时器）；触发成功后立即静默补刷 + 800ms 再补一次；`pollBuilds`/`selectJob` 命中 settled / expired / stale 即清空跟踪；切视图、卸载时清空。

## 视图改名（2026-10-22）

方案与接口实测见 [`docs/plan-view-rename.md`](./plan-view-rename.md)。

红（stub 阶段，6 例后端 + 2 例前端）：

```
--- FAIL: TestRenameViewPostsFullForm (0.00s)
    update_test.go:375: form name = "", want Builds2
    update_test.go:378: form item_demo = "", want on
    update_test.go:396: restore path = "", want /view/Builds2/config.xml
--- FAIL: TestRenameViewIncludesRegexInForm (0.00s)
    update_test.go:430: useincluderegex missing: Jenkins would drop includeRegex
--- FAIL: TestRenameViewRejectsDuplicateName (0.00s)
    update_test.go:458: error = <nil>, want invalid_config
--- FAIL: TestRenameViewRejectsNonListView (0.00s)
    update_test.go:489: error = <nil>, want invalid_config
--- FAIL: TestUpdateViewRenamesThenPatchesNewPath (0.00s)
    update_test.go:542: UpdateView() error = 404 page not found (HTTP 404)
--- FAIL: TestUpdateViewRenameOnlyWritesOnce (0.00s)
    update_test.go:588: UpdateView() error = 404 page not found (HTTP 404)

FAIL  src/lib/forms.test.ts > buildViewUpdateFields > detects a renamed view
AssertionError: expected {} to deeply equal { name: 'Builds 2' }
FAIL  src/lib/forms.test.ts > buildViewUpdateFields > trims the new name before comparing
```

RPC 层红（mock 缺 `/api/json` 与 `configSubmit`）：

```
--- FAIL: TestRPCUpdateViewRename (0.00s)
    config_rpc_test.go:393: updateView error = not_found: 404 page not found
```

绿：

```
ok  github.com/nixing178/dbx-plugin-jenkins/internal/jenkins  1.471s
ok  github.com/nixing178/dbx-plugin-jenkins/internal/rpc      0.793s

 Test Files  11 passed (11)
      Tests  129 passed (129)

pnpm typecheck: 通过
pnpm build: [check-version] ok: 0.4.0 → ui/index.html 201.06 kB
```

真实 Jenkins 2.586 E2E：

```
CRUD view updated: "updated by dbx" demo,param-demo
CRUD view renamed: CRUD-View-Renamed "updated by dbx" demo,param-demo
CRUD cleanup ok
E2E OK
```

> 改名后描述（`updated by dbx`）与任务清单（`demo,param-demo`）都保留，说明 configSubmit 的完整表单 + XML 回写生效。

新增覆盖：

- `RenameView`：表单必含 `name`、`json`（description/recurse/filterQueue/filterExecutors/columns）与每个任务的 `item_<job>`；有正则时补 `useincluderegex` + `includeRegex`；改名后用原 XML 回写新路径；
- 重名（区分大小写）不发任何写请求，返回 `invalid_config`；
- 非 ListView（AllView）拒绝改名，返回 `invalid_config: view is not a ListView`；
- `UpdateView`：先改名再读新路径 XML 做局部改写，返回最终名称；只改名时只写一次 XML；同名视为 no-op；
- RPC `jenkins/updateView` 返回 `fullName`；
- 前端 `buildViewUpdateFields` 比对 trim 后的名称，空名等于不修改。

## 覆盖点

- `BaseURL`：scheme/host/port 校验、base_path 归一化、IPv6、非法输入。
- `JobPath`：嵌套文件夹、URL 转义、空段与 `..` 拒绝。
- `NormalizeColor`：6 种状态 + `_anime` + 大小写/空白。
- `Client`：Basic Auth、匿名、CookieJar、CSRF 自动获取与 403 重试一次、CSRF 关闭降级、只读守卫、401/403/404/500 映射、超时、自签证书开关。
- `RPC`：`connection/test|connect|disconnect`、5 种 secret 传递形态、base_path、只读标记、幂等连接/断开、会话隔离、未知连接提示。
- `Jobs`：根/文件夹列表、文件夹识别与排序、浮层 fullName 补全、状态归一化、参数定义（string/choice/boolean）。
- `Builds`：构建列表默认 limit、触发原因、构建详情、变更集、产物、结果→状态。
- `Console`：`progressiveText` 增量、`X-Text-Size`/`X-More-Data`、缺失 header 的兜底、512 KiB 分块。
- `Actions`：无参/带参触发路径、队列 ID 解析、停止构建、只读拦截、HTTP 错误映射。
- `Queue`：队列项解析（task/why/stuck/inQueueSince/cause）、取消队列项、只读拦截。
- `Nodes`：在线/离线、执行器 busy/total、标签、离线原因。
- `Overview`：whoami + mode + queue + nodes 聚合、执行器/节点统计。
- `MCP`：10 个工具定义与 schema、只读连接隐藏 3 个写工具、注解（readOnly/destructive/idempotent）、`mcp/call` 路由、生命周期兜底建连、工具级错误（`isError`）、控制台日志两段返回（正文 + nextStart/moreData）。
- `Views`：`GetViews`、`GetViewJobs`（视图名 URL 转义）、视图内文件夹/任务归一化与排序；RPC folder 优先于 view；MCP `list_jobs` 支持 view；前端 `groupJobs` 拆分与视图 Tab；视图改名（configSubmit 完整表单 + XML 回写、查重、非 ListView 拒绝）。
- `Stages`：`wfapi/describe` 解析、矩阵列顺序、缺失 stage 留空、平均耗时（失败/进行中排除）、提交数、404 → `stages_unavailable`、部分构建无阶段时降级；变更集同时解析 `changeSet`（Freestyle）与 `changeSets`（Pipeline），按 commitId 去重，矩阵内单条信息截断 200 rune。
- `CRUD`：名称校验（路径分隔符/`..`/控制字符/255 上限）；任务 config.xml 路径；自由风格与 Pipeline 最小模板（脚本转义）；文件夹内创建；`doRename`；`doDelete`；ListView `createView`（name/mode 在 form 与 json 各一份、7 列默认配置）；视图 config.xml 读写；9 个写入口只读拦截；XML 保守改写保留 SCM/构建步骤/comparator/columns；增量更新只提交变化字段；HTML 错误页摘要。
- 前端：错误码 → 中英文提示、未知错误码保留 detail；状态色调、时长/时间格式、日志尾部截断、匹配计数、大小写高亮；表单名称校验、载荷裁剪、差异字段、任务列表集合比较、删除确认精确匹配。
