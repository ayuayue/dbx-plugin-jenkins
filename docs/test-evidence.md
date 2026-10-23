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
| PG1 | 构建列表进度列：`estimatedDuration` 透出 + 进度算法（构建中百分比 / 超估算封顶 99% / 缺估算走不确定态 / 已完成返回 null） | `go test ./internal/jenkins/ -run TestGetBuildsReadsEstimatedDuration` + `npx vitest run src/lib/progress.test.ts` | Go：`builds[0].EstimatedDuration undefined (BuildSummary has no field)` 编译失败；前端：4 FAIL / 5（空实现返回 null） | Go 全量 OK；vitest 206 PASS（15 files）；`vue-tsc --noEmit` exit 0；`pnpm build` 产物 `ui/index.html` 223 kB（版本 0.4.4 一致） |
| PG2 | 进度条按阶段步进：单构建 `jenkins/buildStages` + `stageProgress`（已完成一格 / 进行中半格 / 封顶 99 / 空阶段回退时长估算） | `go test ./internal/rpc/ -run TestRPCBuildStages` + `npx vitest run src/lib/progress.test.ts` | Go：`method not found: jenkins/buildStages`（3 FAIL）；前端：`stageProgress is not a function`（5 FAIL / 13） | Go 全量 OK；vitest 214 PASS（15 files）；`vue-tsc --noEmit` exit 0；`pnpm build` 224.07 kB；预览 `#632` 时长估算 63% → 阶段步进 37%，确认阶段优先 |
| EX1 | 构建执行状态列表（0.5.0）：`jenkins/executors` 槽位明细（空闲占位 / Queue$Executable 等待启动 / 节点离线 / 进度复用时长估算） | `go test ./internal/jenkins/ -run "TestGetExecutors\|TestJobNameFromDisplayName"` + `npx vitest run src/lib/executors.test.ts` | Go：`client.GetExecutors undefined`、`undefined: jobNameFromDisplayName`（build failed）；前端：`./executors` 模块缺失、`hasRunningExecutor is not a function`、`expected ['overview'] to deeply equal ['overview','executors']`（3 files / 4 tests FAIL） | Go 全量 OK；vitest 244 PASS（17 files）；`vue-tsc --noEmit` exit 0；`[check-version] ok: 0.5.0`、`pnpm build` 230.47 kB |

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
ok  github.com/nixing178/dbx-plugin-jenkins/internal/jenkins
ok  github.com/nixing178/dbx-plugin-jenkins/internal/rpc

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

### AT：新建任务后视图任务列表不刷新（0.4.1）

红（先写测试，`src/lib/viewRefresh.ts` 尚未实现）：

```
FAIL  src/lib/viewRefresh.test.ts [ src/lib/viewRefresh.test.ts ]
Error: Failed to load url ./viewRefresh (resolved id: ./viewRefresh) in E:/DBX插件开发/Jenkins/src/lib/viewRefresh.test.ts. Does the file exist?
 Test Files  1 failed | 11 passed (12)
      Tests  129 passed (129)
```

绿（实现纯函数 + `App.vue` / `JobList.vue` 接线）：

```
 ✓ src/lib/viewRefresh.test.ts (18 tests)
 Test Files  12 passed (12)
      Tests  147 passed (147)
```

`pnpm typecheck` 无报错；`pnpm build` 通过（`ui/index.html` 202.41 kB）；`go test ./...` 通过；`check-version` 0.4.1 四处一致。

覆盖：

- `isNamedView`：空串 / 纯空白 / `null` / `undefined` / `all` / `ALL` 均非命名视图；首尾空白不影响自定义视图名；
- `containsJob`：按 `fullName` 精确匹配（`demo` 不命中 `demo2`），空名与空列表恒 false；
- `appendJobName`：空集合追加、保持原顺序末尾追加、已存在不重复、`null` 按空集合、输入 trim 与去空白项、空任务名不写入；
- `needsJoinView`：根目录 + 命名视图 + 列表不含新任务才成立；已包含（正则视图）、文件夹内创建、`all`/空视图名、空任务名均跳过；
- `resolveJoinAction`：不需要写回 → `skip`；ListView（含 `jobNames: null`）→ `add`；`fields` 为 `null`（聚合视图等）→ `unsupported`。

UI 接线：`createJob` 先静默刷新列表，再按 `needsJoinView` 读 `viewConfig` 追加 `jobNames` 并回写，最后按结果给三种提示（加入视图 / 普通创建 / 创建但未加入）；`triggerBuild` 触发后与 800ms 补刷各刷一次列表，`settlePendingBuild` 在 `settled` 时刷新一次，`autoRefreshMode` 由 `builds` 转 `null` 时再收敛刷新一次。

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

## 新增任务 400 可读原因（0.4.2）

红（先写测试，6 项 Go + 5 项前端全部失败）：

```
--- FAIL: TestAPIErrorExtractsSendErrorPage
    kind = "request_failed", want rejected
    message = "Jenkins returned an HTML error page (HTTP 400); check the Jenkins server log"
--- FAIL: TestAPIErrorExtractsDuplicateName
--- FAIL: TestAPIErrorStopsAtParagraphBoundary
--- FAIL: TestAPIErrorUnescapesHTMLEntities
--- FAIL: TestAPIErrorKeepsLoggingID
--- FAIL: TestValidateNameRejectsUnsafeCharacters   (14 个字符全部放行)
```

```
 FAIL  src/lib/errors.test.ts > errorKind > reads the kind after an RPC method prefix
 FAIL  src/lib/errors.test.ts > errorText > localizes kinds that carry an RPC method prefix
 FAIL  src/lib/errors.test.ts > errorText > shows the Jenkins reason for rejected requests
 FAIL  src/lib/errors.test.ts > errorText > keeps the detail when a config error explains the problem
 FAIL  src/lib/forms.test.ts > validateName > rejects characters Jenkins refuses
 Tests  5 failed | 28 passed (33)
```

绿：

```
ok  github.com/nixing178/dbx-plugin-jenkins/internal/jenkins
ok  github.com/nixing178/dbx-plugin-jenkins/internal/rpc
```

```
 ✓ src/lib/errors.test.ts (10 tests)
 ✓ src/lib/forms.test.ts (23 tests)
 Test Files  12 passed (12)
      Tests  153 passed (153)
```

`pnpm typecheck` 无报错；`pnpm build` 通过（`ui/index.html` 202.73 kB）；`check-version` 0.4.2 四处一致。

真实 Jenkins 2.586（sidecar 直连，18081 `/jenkins`）：

```
PASS 重名（Jenkins 返回 400，需透出原因）
     -> rejected: A job already exists with the name ‘probe-2’
PASS 名字含 *（本地拦截）        -> invalid_config: name must not contain "*"
PASS 名字含 "（本地拦截，避免 500） -> invalid_config: name must not contain "\""
PASS 中文名                     -> {"fullName":"中文任务-crud","success":true}
PASS 文件夹内新建                -> {"fullName":"probe-folder2/child-b","success":true}
PASS 文件夹内重名                -> rejected: A job already exists with the name ‘child-b’
```

覆盖：

- `newAPIError`：400（非 401/403/404）→ `rejected`；HTML 页提取 `<h1>Error</h1><p>原因`（`<p>` 未闭合，按 `</p>`/`<!--`/`</div>`/`<h1`/`<div` 截断）；HTML 实体反转义（`&lt;` → `<`）；无原因的 500 Oops 页保留通用提示并附 `Logging ID=`。
- `ValidateName`：拒绝 `* : < > | & ; ! @ $ ^ [ ] "`；放行 `( ) { } = + , ' ~ \` - 。` 空格与中文。
- `errorKind` / `errorText`：识别 `jenkins/createJob: rejected: …` 形式的方法前缀；`rejected` 与 `invalid_config` 把 detail 拼在本地化文案后；未知 kind 仍只保留 detail。
- `validateName`（前端）：与后端字符集一致。
- `JobEditor.submit`：`busy` 期间直接返回，避免重复提交导致第二次重名 400。

## 操作提示改 toast（2026-10-22）

顶部内联绿条（`.banner.notice`）改为右上角浮层 toast：`src/lib/toast.ts`（队列：4s 过期、同文案去重重置计时、最多 3 条、点击关闭）、`src/components/ToastHost.vue`（浮层渲染）、App.vue 的 14 处 `showNotice` 全部改走 `showToast`。错误横幅 `.banner.error` 保留（需要常驻 + 重试按钮）。

红（`toast.ts` 未创建时）：

```
❯ src/lib/toast.test.ts (0 test)
Error: Failed to load url ./toast (resolved id: ./toast) ... Does the file exist?
 Test Files  1 failed | 12 passed (13)
      Tests  153 passed (153)
```

第一次写实现后仍红 1 例——不是实现缺陷，是测试过度约束了「两条同时到期各回调一次」：

```
FAIL  src/lib/toast.test.ts > createToastQueue > notifies subscribers on push, drop and expiry
AssertionError: expected "spy" to be called 3 times, but got 4 times
      Tests  1 failed | 165 passed (166)
```

绿：

```
 ✓ src/lib/toast.test.ts (13 tests)
 Test Files  13 passed (13)
      Tests  166 passed (166)
```

`pnpm typecheck` 无报错；`pnpm build` 通过（`ui/index.html` 213.76 kB）；`check-version` 0.4.2 四处一致。

UI 目视（Chrome headless 截图，`pnpm dev` + `/toast-preview.html`）：

- `docs/toast-preview.png`：浅色主题，三条堆叠（success 绿左条 / info 蓝左条 / 长文案换行），浮层压在 56px 顶栏下方、不挤压列表。
- `docs/toast-preview-dark.png`：深色主题，卡片随 `--panel/--text` 变量翻转，文字对比正常。

覆盖点：

- 入队返回 id、默认 success、info 保留 tone、文本 trim、空文本返回 null。
- 4s 自动过期；同文案重复 push 不叠加且重置倒计时；过期后可再次入队。
- 超过 3 条挤掉最早一条；`dismiss` 立即移除并清定时器；未知 id 不影响其它条目。
- `clear()` 清空并取消所有计时器；`onChange` 在入队/挤出/过期/清空时回调。

---

## 2026-10-22 · 自动刷新治理（0.4.3）

范围：`docs/refresh-hardening-plan.md` 的 R1~R10，决策 D1:A / D2:A / D3:A / D4:A / D5:A。

### Step 1 轮询基建（R3/R4/R5）

红（`POLL_TIMEOUT_MS` / 护栏 / `trigger()` 尚未实现）：

```
 ❯ src/lib/refresh.test.ts (38 tests | 4 failed)
   × refresh intervals > keeps poll requests on a shorter timeout than manual loads
     → expected undefined to be 15000
   × createAutoRefresher failure guard > stops once consecutive failures reach the limit
     → expected true to be false
   × createAutoRefresher trigger > runs immediately and resets the countdown
     → refresher.trigger is not a function
   × createAutoRefresher trigger > does nothing on an inactive refresher
     → refresher.trigger is not a function
      Tests  4 failed | 34 passed (38)
```

绿：

```
 ✓ src/lib/refresh.test.ts (38 tests)
      Tests  38 passed (38)
```

### Step 2 构建详情轮询（R1）

红：

```
   × shouldPollBuildDetail > returns false for a missing build
   × shouldPollBuildDetail > polls while the build is running
   × shouldPollBuildDetail > detects the building status without the flag
   × shouldPollBuildDetail > stops once the build finished
     → shouldPollBuildDetail is not a function
      Tests  4 failed | 38 passed (42)
```

绿：`✓ src/lib/refresh.test.ts (42 tests)`。

### Step 3 队列轮询（R2）

红：`expected undefined to be 5000 // QUEUE_AUTO_REFRESH_INTERVAL_MS`（1 failed | 42 passed）。
绿：`✓ src/lib/refresh.test.ts (43 tests)`。

### Step 4 延迟刷新收敛（R6/R9/R10）

红：

```
   × shouldRefreshJobsAfterPending > refreshes once the new build showed up
   × shouldRefreshJobsAfterPending > refreshes when the tracking window expired
   × shouldRefreshJobsAfterPending > skips while still watching or after a job switch
     → shouldRefreshJobsAfterPending is not a function
      Tests  3 failed | 43 passed (46)
```

绿：`✓ src/lib/refresh.test.ts (46 tests)`，全仓 180 passed。

### Step 5 控制台内存与后台（R8）

红：

```
 ❯ src/lib/console.test.ts (10 tests | 4 failed)
   × trimLogTail > leaves short logs untouched
   × trimLogTail > treats a non-positive limit as unlimited
   × trimLogTail > keeps the tail and drops the partial first line
   × trimLogTail > falls back to the plain tail when there is no line break
     → trimLogTail is not a function
      Tests  4 failed | 6 passed (10)
```

绿：`✓ src/lib/console.test.ts (10 tests)`。

### 最终验收

```
 Test Files  13 passed (13)
      Tests  184 passed (184)
```

```
> node scripts/check-version.mjs && vue-tsc --noEmit && vite build
[check-version] ok: 0.4.3
✓ 71 modules transformed.
ui/index.html  216.89 kB │ gzip: 68.53 kB
✓ built in 1.69s
```

新增覆盖点：

- 连续失败达阈值后自动停轮询并回调 `onPaused("failures")`；中间成功一次即清零；未配置阈值时保持旧行为（无限重试）。
- `trigger()` 立即执行一轮并重置倒计时；未启动时为空操作。
- `shouldPollBuildDetail`：`building` 真值 / `status=building` / 缺省 / 已结束。
- `shouldRefreshJobsAfterPending`：settled 与 expired 都补刷，watching 与 stale 不刷。
- `trimLogTail`：未超限原样返回、超限按行边界留尾、无换行退化为纯尾、非正数视为不限。
- `QUEUE_AUTO_REFRESH_INTERVAL_MS = 5000`；`POLL_TIMEOUT_MS = 15000`。

无法用纯函数覆盖的行为（已人工审查）：`visibilitychange` 隐藏停轮询/回前台 `trigger()`、`stopBuild` 延迟刷新目标校验、`refreshJob` 的 `restart()` 前置、`pollBuilds` 的 `selectToken` 快照校验、ConsoleView 的 `inFlight` 代际互斥。

---

## 2026-10-22 · 自动刷新提示与页面反馈（0.4.4）

范围：`docs/refresh-feedback-plan.md` 的 F1~F6，决策 D1:A / D2:A / D3:A / D4:A / D5:A。

### Step 1 轮询器运行回调

红（`onRunStart` / `onRunEnd` 尚未接）：

```
 ❯ src/lib/refresh.test.ts (49 tests | 3 failed)
   × createAutoRefresher run hooks > reports the start and a successful end of each run
     → expected [] to deeply equal [ 'start', 'ok' ]
   × createAutoRefresher run hooks > reports a failed run and still counts it towards the guard
     → expected [] to deeply equal [ false, false ]
   × createAutoRefresher run hooks > still reports the end of an in-flight run after stop()
     → expected [] to deeply equal [ true ]
      Tests  3 failed | 46 passed (49)
```

绿：`✓ src/lib/refresh.test.ts (49 tests)`。

> 顺带修正：四个 `pollXxx` 原先在内部 `catch` 掉错误，`AutoRefresher` 永远认为成功——护栏的「连续 3 次失败暂停」此前实际不可能触发。现在错误冒泡到轮询器，计数才生效。

### Step 2 刷新状态纯函数

红（模块缺失）：`Failed to load url ./refreshStatus ... Tests no tests`。
绿：`✓ src/lib/refreshStatus.test.ts (14 tests)`。

### 最终

```
 Test Files  14 passed (14)
      Tests  201 passed (201)
```

```
[check-version] ok: 0.4.4
✓ 75 modules transformed.
ui/index.html  221.19 kB │ gzip: 69.78 kB
```

新增覆盖点：

- `onRunStart` 在 task 前触发、`onRunEnd(true)` 在成功后触发；失败回调 `false` 且仍计入护栏；`stop()` 时仍在飞的请求也会回调 end，界面不会卡在「请求中」。
- `refreshBadge`：暂停优先于请求中；无 mode 时返回 idle（即使 paused 残留也不显示）。
- `formatClockTime`：补零、23 点不溢出、无值返回空串。
- `diffBuildRows`：首次加载不高亮、新增与状态翻转变更、上一次为空列表时高亮、下一次为空时忽略。

目视（Chrome headless，`pnpm dev` 打开 `/refresh-preview.html`）：

- `docs/refresh-badge-preview.png`：浅色下四态与按钮三态，暂停为琥珀色。
- `docs/refresh-badge-preview-dark.png`：深色下文字与圆点对比正常。

## 2026-10-22 · 进入页面自动刷新（0.4.5）

需求：切换页面时自动补刷一次。决策 D1:B / D2:A / D3:A / D4:A / D5:A。

### Step 1 进入动作判定（`src/lib/enterRefresh.ts`）

红（模块缺失）：

```
 FAIL  src/lib/enterRefresh.test.ts [ src/lib/enterRefresh.test.ts ]
Error: Failed to load url ./enterRefresh (resolved id: ./enterRefresh) in src/lib/enterRefresh.test.ts. Does the file exist?
 Test Files  1 failed (1)
      Tests  no tests
```

绿：`✓ src/lib/enterRefresh.test.ts (11 tests)`。

### Step 2 节流语义修正

第一版只按时间戳节流：刚加载完概览（`lastRefreshAt` 已更新）后立刻点「任务」页会被挡掉，列表直接空白。
改成**换页必刷、只有同一页重复进入才节流**（`EnterRefreshGate`）。

红：

```
 ❯ src/lib/enterRefresh.test.ts (12 tests | 4 failed)
   × shouldRefreshOnEnter > 换页面总是刷新，不受上次刷新时间影响
   × shouldRefreshOnEnter > 首次进入某页（还没有上次记录）总是刷新
   × shouldRefreshOnEnter > 同一页刚好到达最小间隔就放行
   × shouldRefreshOnEnter > 支持自定义间隔；间隔为 0 表示不节流
      Tests  4 failed | 8 passed (12)
```

绿：`✓ src/lib/enterRefresh.test.ts (12 tests)`。

### App.vue 接线

- `switchView` → `await refreshOnEnter()`；任务页保留当前 `folder`。
- 阶段视图 `@back` 由裸赋值 `view = 'jobs'` 改为 `backToJobs()`（静默补刷一次）。
- `handleVisibility` 在没有轮询模式时也走 `refreshOnEnter`（概览 / 节点 / 空闲的任务页）。
- `enterRefreshBusy` 防重入；刷新期间用户切走，结束后会给新页补一次。
- 任务页用 `loadJobs(folder, true)` + `selectJob(..., true)` 静默刷新，不闪骨架；用 `selectToken` 快照避免 `loadJobs` 自动重选后重复请求。

### 最终验收

```
 Test Files  16 passed (16)
      Tests  226 passed (226)
```

```
cd backend && go test ./...
ok  github.com/nixing178/dbx-plugin-jenkins/internal/jenkins  1.425s
ok  github.com/nixing178/dbx-plugin-jenkins/internal/rpc      1.682s
```

```
[check-version] ok: 0.4.5
✓ 80 modules transformed.
ui/index.html  224.79 kB │ gzip: 70.79 kB
```

打包与身份：

```
Success: Built ...\dist\io.github.nixing178.jenkins-0.4.5-windows-x64.dbxp
Signature: unsigned review candidate
[verify-sidecar] ok: io.github.nixing178.jenkins/0.4.5
```

新增覆盖点：

- `resolveEnterActions`：概览 / 队列 / 节点各刷自己；任务页首次补 `views`；有选中任务时追加 `detail`；build / stages 返回空；顺序固定 views → jobs → detail。
- `shouldRefreshOnEnter`：换页必刷；首次进入必刷；同页 2s 内跳过、边界放行；自定义间隔与 0 不节流；时钟回拨不刷。

> 说明：App.vue 的分发属接线层，未做 UI 自动化；判定逻辑全部由纯函数单测覆盖，接线通过类型检查 + 构建 + 代码审查确认。

## 构建执行状态列表（0.5.0，2026-10-22）

需求来源：Jenkins 原生首页右栏有 Build Queue 与 Build Executor Status 两个 widget，插件只有前者。决策 D1:1 / D2:1 / D3:1 / D4:2 / D5:1 —— 放概览页、有执行中才 5s 轮询、点行跳任务、不暴露 MCP、版本升 0.5.0。

### 红

```
$ cd backend && go test ./internal/jenkins/ -run "TestGetExecutors|TestJobNameFromDisplayName"
internal\jenkins\system_test.go:156:22: client.GetExecutors undefined (type *Client has no field or method GetExecutors)
internal\jenkins\system_test.go:205:22: client.GetExecutors undefined (type *Client has no field or method GetExecutors)
internal\jenkins\system_test.go:230:13: undefined: jobNameFromDisplayName
FAIL	github.com/nixing178/dbx-plugin-jenkins/internal/jenkins [build failed]
```

```
$ npx vitest run src/lib/executors.test.ts src/lib/refresh.test.ts src/lib/enterRefresh.test.ts
 ❯ src/lib/executors.test.ts (0 test)
   → Failed to load url ./executors ... Does the file exist?
 ❯ src/lib/enterRefresh.test.ts (12 tests | 1 failed)
   × 概览页刷新概览数据 + 执行器列表 → expected [ 'overview' ] to deeply equal [ 'overview', 'executors' ]
 ❯ src/lib/refresh.test.ts
   × TypeError: hasRunningExecutor is not a function
 Test Files  3 failed (3)
      Tests  4 failed | 60 passed (64)
```

### 绿

```
$ cd backend && go test ./...
ok  github.com/nixing178/dbx-plugin-jenkins/internal/jenkins  1.246s
ok  github.com/nixing178/dbx-plugin-jenkins/internal/rpc      0.865s
```

```
$ npx vitest run
 ✓ src/lib/executors.test.ts (15 tests)
 ✓ src/lib/refresh.test.ts (52 tests)
 ✓ src/lib/enterRefresh.test.ts (12 tests)
 Test Files  17 passed (17)
      Tests  244 passed (244)
```

```
$ pnpm build
[check-version] ok: 0.5.0
✓ 84 modules transformed.
ui/index.html  230.47 kB │ gzip: 72.14 kB
```

### 实现要点

| 位置 | 处理 |
| --- | --- |
| `backend/internal/jenkins/system.go` | `GetExecutors` 复用 `/computer/api/json`，独立 tree（不动 `GetNodes`，避免概览聚合回归）；执行器数组下标 +1 作为槽位序号；空闲槽位也返回 |
| 同上 | `jobNameFromDisplayName` 剥掉 `Run.getFullDisplayName()` 尾部的 ` #N`；自定义 displayName 不遵循该格式时原样返回 |
| 同上 | `Queue$Executable`（占位但构建未建）只带 url → `pending=true`，前端显示「等待启动」而不是报错 |
| `src/lib/executors.ts` | 纯函数：`executorState`（idle/pending/building/settling）、`executorSummary`（`!idle` 计 busy）、`executorProgress` 复用 `buildProgress`、`executorNodeLabel`（空 displayName → 内置节点） |
| `src/lib/refresh.ts` | `hasRunningExecutor` 只看 `idle === false`；`EXECUTOR_AUTO_REFRESH_INTERVAL_MS = 5000` |
| `src/App.vue` | `AutoRefreshMode` 新增 `executors`，仅概览页且有占用时启用；`pollExecutors` 不内部 catch，错误上抛给护栏计数 |

### 覆盖点

- 后端：三槽位（内置节点构建中 / 内置节点空闲 / 离线 agent 构建中）字段与顺序；`pending` 占位形态；`jobNameFromDisplayName` 五种输入（含中文与空格、number=0、空串）。
- 前端：四种执行器状态；`busy/total` 汇总（含 pending 算 busy、空列表不出 NaN）；进度（45% / 超时封顶 99 / 缺估算不定态 / 空闲与等待启动返回 null）；`#N` 标签；轮询判定（占用、等待启动、全空闲、空列表、未加载）。
- 进入刷新：概览页 actions 变为 `["overview", "executors"]`，先数字后明细；其余页不变。

> 说明：App.vue 的分发属接线层，判定逻辑全部由纯函数单测覆盖，接线通过 `vue-tsc --noEmit` + 构建确认。

### 预览（`docs/executor-preview*.png`）

- `pnpm dev` 打开 `/executor-preview.html`（`?theme=dark` 切深色），假数据覆盖六种形态：内置节点构建中 61%、等待启动、空闲、缺估算的不定态进度、收尾中、离线节点。
- 截图：`docs/executor-preview.png`、`docs/executor-preview-dark.png`（Chrome headless，1360 宽）。
- 深色首轮截图暴露「构建中」标签用 `var(--accent)` 作字色对比不足（`#3355ff` on `#1c1f26`）→ 改为「底色 + 1px 描边表达色相、文字统一用 `var(--text)`」；`--warning` 同理，不再硬编码 `#d99b1a1f` / `#b47d0d`。

### 打包与冒烟（0.5.0）

```
$ npx --yes @dbx-app/plugin-cli@0.1.9 package .
Building: Go backend
Packaging: ...\dist\io.github.nixing178.jenkins-0.5.0-windows-x64.dbxp
Success: Built ...\dist\io.github.nixing178.jenkins-0.5.0-windows-x64.dbxp
Signature: unsigned review candidate
```

包内 6 条目；`ui/index.html` 232,537 B 与本地逐字节一致；manifest 0.5.0；sidecar 8,574,976 B → `[verify-sidecar] ok: io.github.nixing178.jenkins/0.5.0`。

从 .dbxp 取出 sidecar 起 mock Jenkins 连接冒烟（临时脚本已删）：

```
[smoke] jenkins/executors -> 3 rows
  node="" #1 idle=false pending=false job="demo-job" build=12 building=true
  node="" #2 idle=false pending=true job="" build=0 building=false
  node="agent-1" #1 idle=true pending=false job="" build=0 building=false
[smoke] jenkins/nodes -> built-in busy=2/2
[smoke] ok: executors 3 rows + nodes 未受影响
```

确认：空闲槽位占位、`Queue$Executable` → `pending`、离线节点标记、`GetNodes` 的 busy 统计未被新 tree 影响。

## 修复：执行器行跳转 404（0.5.1，2026-10-22）

### 根因

`Run.getFullDisplayName()` 返回的是 `project.getFullDisplayName() + ' ' + displayName`，而

```java
// hudson.model.AbstractItem
public final String getFullDisplayName() {
    return getParent().getFullDisplayName() + " » " + getDisplayName();   // 分隔符是 » 不是 /
}
```

所以文件夹里的任务拿到的是 `Builds » demo-web #12`，按 ` #N` 剥出来是 `Builds » demo-web`，
拼成 `/job/Builds » demo-web` 必然 404（界面显示「Jenkins 地址、上下文路径或资源不存在」）。
自定义 displayName 的任务同样对不上。

### 修复

改为从 `currentExecutable.url` 反解 fullName（`Run.getUrl()` = `project.getUrl() + number + "/"`，
`project.getUrl()` 是 `/job/<folder>/job/<job>/`），新增 `jobFullNameFromURL`：

```go
segments := strings.Split(strings.Trim(parsed.Path, "/"), "/")
for i := 0; i < len(segments)-1; i++ {
    if segments[i] == "job" { names = append(names, segments[i+1]); i++ }
}
```

`jobNameFromDisplayName` 降级为 url 缺失时的兜底，注释里写明只对顶层任务可靠。

### 红 → 绿

```
$ go test ./internal/jenkins/ -run "TestGetExecutors|TestJobFullNameFromURL"
internal\jenkins\system_test.go:236:13: undefined: jobFullNameFromURL
FAIL	[build failed]

$ go test ./...
ok  internal/jenkins  1.176s
ok  internal/rpc      0.750s
```

`TestGetExecutors` 的 mock 换成真实形态（`Builds » demo-web #12` + 编码过的 folder url），
新增 `TestJobFullNameFromURL` 10 个用例：顶层 / context path / 文件夹 / 三层 / 相对地址 /
URL 编码中文 / 缺尾斜杠 / 带查询串 / 队列占位 / 空串。

### 打包与端到端冒烟（0.5.1）

- `dist/io.github.nixing178.jenkins-0.5.1-windows-x64.dbxp`，4,954,841 B，sha256 `a1221d54…`；
  `ui/index.html` 与本地一致；`[verify-sidecar] ok: io.github.nixing178.jenkins/0.5.1`。
- mock Jenkins（`fullDisplayName` 用 `»`、url 用 folder 结构）→ 从包内 sidecar 走完整链路：

```
[smoke] executors[0]: jobName="Builds/demo-web" build=12
[smoke] jenkins/job -> Builds/demo-web (status building)
[smoke] jenkins/build -> #12
[smoke] ok: 文件夹任务名从 url 反解正确，跳转链路 200
```

## 构建详情页变更记录列表优化（0.6.0，2026-10-22）

### 需求与决策

源哥一句话「构建任务详情页-变更记录列表优化」，按 `docs/plan-build-changes.md` 推荐项全取：
D1:2 展示层 + 数据增强、D2:A 时间 + 文件数 + 正文、D3:2 最新在前、D4:1 折叠 8 条、
D5:1 点哈希复制、D6:1 首行 + 展开正文、D7:1 不做搜索、D8:1 浮层只复用排序、D9:1 0.6.0、D10:1 MCP 透出。

### 真机字段（临时 Jenkins 2.587，git-pipeline #3）

```
GET /job/git-pipeline/3/api/json?tree=changeSets[items[commitId,msg,comment,timestamp,date,author[fullName],affectedPaths]]
{"items":[{"commitId":"8e45f64…","msg":"probe B: 第二条提交",
           "comment":"probe B: 第二条提交\n\n- 明细一…\n- 明细二…\n",
           "timestamp":1792659786000,"date":"2026-10-22 17:03:06 +0800",
           "author":{"fullName":"e2e"},"affectedPaths":["PROBE.md"]}]}
```

四个必须记住的点：

1. **`msg` 只有首行（subject）**，多行正文在 `comment`；原来的详情页不是"多行撑爆"，而是"正文根本没取"。
2. **`date` 是字符串、`timestamp` 才是 epoch 毫秒** → 由后端归一化，前端不碰 `date`。
3. **tree 写 `items[*]` 会让 `author` 变成 `{}`**（本次实测踩到），必须显式 `author[fullName]`。
4. changelog 顺序是**旧→新**，且同秒提交 timestamp 相同 → 前端排序必须稳定。

### 后端红 → 绿

```
$ go test ./internal/jenkins/ -run "TestGetBuildDetailReadsChangeDetails|TestGetBuildChangeTimestampFallsBackToDate|TestGetBuildChangeBodyEmptyForSingleLine|TestGetStageMatrixDropsChangeBody"
internal\jenkins\builds_test.go:227:12: change.Timestamp undefined (type ChangeSetItem has no field or method Timestamp)
internal\jenkins\builds_test.go:230:12: change.FileCount undefined ...
FAIL	[build failed]

$ go test ./...
ok  	internal/jenkins	1.300s
ok  	internal/rpc	0.917s
```

新增 4 例：详情读全字段（含 tree 断言）/ `date` 兜底归一化 / 单行提交无正文 / 矩阵不请求 comment 且 Body 为空。

一次假绿：`changeBody` 起初先 `TrimSpace` 再切行，单行 `comment` 的换行被吃掉，
`TestGetBuildChangeBodyEmptyForSingleLine` 报 `body = "fix bug"` → 改成先按原始文本切行再修剪。

### 前端红 → 绿

```
$ npx vitest run src/lib/changes.test.ts
Error: Failed to load url ./changes (resolved id: ./changes) ... Does the file exist?
Test Files  1 failed (1)   Tests  no tests

$ npx vitest run
Test Files  18 passed (18)
     Tests  263 passed (263)
```

`src/lib/changes.ts` 19 例：稳定倒序 / 时间全缺失保序 / 部分缺失 / 同秒稳定 / 不改原数组 /
折叠边界（8、全量、不足 8、limit≤0）/ 摘要计数（空作者、缺 fileCount）/ 正文修剪 /
同年 `10-22 17:03` / 跨年 `2025-12-31` / 英文 `Oct 22, 17:03` / 缺失 `—`。

### 真机端到端（真实 client 直连，探针已删）

```
$ go run ./cmd/probe
[
  {"commitId":"6a24159…","message":"probe A: 第一条提交","author":"e2e","timestamp":1792659786000,"fileCount":1},
  {"commitId":"8e45f64…","message":"probe B: 第二条提交","author":"e2e","timestamp":1792659786000,"fileCount":1,
   "body":"- 明细一：验证多行消息的展示\n- 明细二：应默认折叠为单行摘要\n- 明细三：展开后才看到全部内容"},
  {"commitId":"0c69226…","message":"probe C: 第三条提交","author":"e2e","timestamp":1792659786000,"fileCount":1}
]
```

- `timestamp` 与 `git log --format=%ai` 一致（2026-10-22 17:03:06 +0800）；
- `body` 只出现在多行提交上、首行已剥掉；单行提交不带 `body` 字段（omitempty）。

### 交互（CDP，1280×980）

```
初始（折叠 8 条）：rows=8,  moreLabel="展开全部 12 条", bodyCount=0, summary="12 个提交 · 3 位作者 · 98 个文件"
点击展开全文：      rows=8,  bodyCount=1, toggle="收起全文"
点击展开全部：      rows=12, moreLabel="收起"
最后一行时间：      2025-10-07（跨年显示完整日期）
点击收起：          rows=8
```

### 预览

- `docs/screenshots/build-changes-preview.png`（浅色）
- `docs/screenshots/build-changes-preview-dark.png`（深色）
- `docs/screenshots/build-changes-expanded.png`（展开全部 + 正文展开）
- 页面：`changes-preview.html` + `src/changes-preview-main.ts`（`pnpm dev` 打开，`?theme=dark` 切深色）

### 验证命令汇总

```
go test ./...              # 全绿
npx vitest run             # 263 PASS（18 files）
npx vue-tsc --noEmit       # 0 错
pnpm build                 # [check-version] ok: 0.6.0，ui/index.html 235.25 kB
```

### 打包（0.6.0）

- `dist/io.github.nixing178.jenkins-0.6.0-windows-x64.dbxp`，unsigned review candidate。
- 包内 6 条目；`ui/index.html` 237,455 B 与本地逐字节一致（sha256 `75c6082b…`）。
- 从包内提取 `bin/windows-x64/dbx-plugin-jenkins.exe`（8,579,072 B）→
  `[verify-sidecar] ok: io.github.nixing178.jenkins/0.6.0`。

---

## 0.7.0 · 控制台日志列表（2026-10-23）

### 红 → 绿

新增 `src/lib/logLines.test.ts`（27 例），先跑出模块缺失红：

```
FAIL src/lib/logLines.test.ts [ src/lib/logLines.test.ts ]
Error: Failed to load url ./logLines ... Does the file exist?
Test Files 1 failed (1) / Tests no tests
```

实现后一次断言修正：`ESC[22m` 只取消加粗、不重置前景色，绿色应延续（测试期望写错，非实现问题）。

```
✓ src/lib/logLines.test.ts (27 tests) 8ms
```

覆盖：ANSI CSI / OSC / 光标序列剥离、颜色与 bold 合并、未知 SGR 忽略、
级别判定正反例（`0 errors` / `errorRate` 不误报）、过滤保号、命中行下标、行号起点。

### 全量

```
vitest run     # 290 PASS（19 files）
vue-tsc        # 0 错
go test ./...  # ok（jenkins / rpc）
pnpm build     # [check-version] ok: 0.7.0，ui/index.html 245.33 kB
```

### 交互（CDP，1280×1000，`?big=1` 共 6200 行）

```
初始：      rows=5000, first=1021, hint="仅显示最后 5000 行（行号为显示序号）。",
            footer="6020 行 · 显示 5000 行"
仅问题：    rows=1048, levels=[error,warn]
搜索：      flaky-retry → 1/117，active=1032
下一处：    2/117，active=1075（偏移 43 = 每轮行数）
全屏：      fullscreen=true，log-body clientHeight=895
Esc：       fullscreen=false
DOM 顺序：  严格递增（sorted=true），单行高度 22px
```

### 预览

- `docs/screenshots/console-list-preview.png`（浅色，42 行）
- `docs/screenshots/console-big-default.png`（6200 行截断 + 折行悬挂缩进）
- `docs/screenshots/console-big-problems.png`（仅问题）
- `docs/screenshots/console-big-search.png`（搜索命中 + 自动取消跟随 + ↓ 最新）
- `docs/screenshots/console-big-fullscreen.png`
- 页面：`console-preview.html` + `src/console-preview-main.ts`（`?big=1` / `?theme=dark`）

### 打包（0.7.0）

- `dist/io.github.nixing178.jenkins-0.7.0-windows-x64.dbxp`，4,961,298 B，unsigned review candidate。
- sha256 `3669bb85924e175e…`。
- 包内 6 条目；`ui/index.html` 247,690 B 与本地逐字节一致（sha256 `ef9400e1f2f50248…`）。
- 从包内提取 `bin/windows-x64/dbx-plugin-jenkins.exe`（8,579,072 B）→
  `[verify-sidecar] ok: io.github.nixing178.jenkins/0.7.0`。

---

## 0.7.1 · 控制台日志区高度（2026-10-23）

- 默认高度 `clamp(360px, 52vh, 720px)` → `clamp(420px, 64vh, 900px)`；
  1000px 高窗口下 520px → 640px，可视行数约 +5 行。
- `vue-tsc` 0 错；`pnpm build` → `[check-version] ok: 0.7.1`，`ui/index.html` 247,690 B
  （与 0.7.0 同大小，仅 CSS 数字等长替换），产物内含 `clamp(420px,64vh,900px)`。
- 截图：`docs/screenshots/console-height-preview.png`（6200 行）。
- 打包：`dist/io.github.nixing178.jenkins-0.7.1-windows-x64.dbxp`，4,961,304 B，
  sha256 `fc51b5a75f930e41…`，包内 ui 与本地逐字节一致，
  sidecar `[verify-sidecar] ok: io.github.nixing178.jenkins/0.7.1`。

---

## 0.7.2 · 首屏默认滚到底（2026-10-23）

### 根因

1. 首屏 5000 行带 `content-visibility: auto`，`scrollHeight` 先按 `contain-intrinsic-size`
   估算，一次 `scrollTop = scrollHeight` 后真实行高撑开，落点就不在底部了；
2. 程序化滚动也会触发 `scroll` 事件，布局抖动瞬间 `distance > 48` 会被 `handleScroll`
   误判成「用户上翻」，把 `follow` 关掉，后续追加不再跟随。

### 修复

- `scrollToBottom()` 改为三帧 settle：每帧重设 `scrollTop`，直到距离收敛；
- 新增 `scrollGuard` 代际，程序滚动期间 `handleScroll` 直接返回；
- `reset()` 恢复 `follow = true`（上一次手动上翻不影响新打开的构建）。

### 验证

```
首屏：      distance=0, scrollTop=109309, follow=true,  latestVisible=false
上翻 4 档：  distance=1600, follow=false, latestVisible=true
点「最新」： distance=0, follow=true,  latestVisible=false
vitest      # 290 PASS（19 files）
vue-tsc     # 0 错
pnpm build  # [check-version] ok: 0.7.2，ui/index.html 245.50 kB
```

截图：`docs/screenshots/console-first-open-bottom.png`。

### 打包（0.7.2）

- `dist/io.github.nixing178.jenkins-0.7.2-windows-x64.dbxp`，4,961,399 B，unsigned review candidate。
- sha256 `b23d72ffe8834b40…`；包内 6 条目，`ui/index.html` 247,856 B 与本地逐字节一致。
- sidecar `[verify-sidecar] ok: io.github.nixing178.jenkins/0.7.2`。

---

## 0.7.3 · 日志区底部被裁（2026-10-23）

### 根因

`BuildView` 是 `display:flex; flex-direction:column`，`.console` 作为 flex item 默认
`flex-shrink: 1`。页面内容超过视口时，面板被压到 318px，而 `.log-body` 的
`min-height: 260px` 顶住不让缩 → 工具栏 + 日志 + footer 总高 359px 溢出，
`.console { overflow:hidden }` 把底部裁掉，**footer 完全不可见**。

复现（860px 高视口，`?layout=build&big=1`）：

```
修复前：console.clientHeight=318, scrollHeight=359
        log-body.bottom=820 > console.bottom=806
        footer: top=820 bottom=846 visible=false
修复后：console.clientHeight=611, scrollHeight=611
        footer: top=1072 bottom=1098 visible=true
        distance=0, lastFullyVisible=true
```

窄窗复测（760×620）：`console 553 = scrollHeight 553`，footer `visible=true`。

### 修复

`ConsoleView.vue` 的 `.console` 加 `flex: 0 0 auto`（不参与收缩，超出交给
`.build-view` 滚动）。全屏时元素是 `position: fixed`，不受影响。

### 验证

```
vitest      # 290 PASS（19 files）
vue-tsc     # 0 错
pnpm build  # [check-version] ok: 0.7.3，ui/index.html 245.51 kB
```

截图：`docs/screenshots/console-clip-after.png`（修复前状态代码已变更无法复现，此处只保留修复后）。

### 打包（0.7.3）

- `dist/io.github.nixing178.jenkins-0.7.3-windows-x64.dbxp`，4,961,406 B，unsigned review candidate。
- sha256 `b6c69ea085edac65…`；包内 6 条目，`ui/index.html` 247,870 B 与本地逐字节一致。
- sidecar `[verify-sidecar] ok: io.github.nixing178.jenkins/0.7.3`。

---

## 0.7.4 · 滚到底看不到最后几行（2026-10-23）

### 现象

真机 1006 行日志，滚动条已到底、`自动跟随` 勾选着，但最后 8 行（999–1006）
没显示，末行 998 还被裁掉一半。

### 根因

`.log-row` 上的 `content-visibility: auto` + `contain-intrinsic-size: auto 22px`：
未进入视口的行按 22px 估算，而折行行实际 44/66px。滚动高度因此被低估，
滚动条停在「估算底部」，真实末尾还在下面。

实测（`?long=1`，5000 行、混入 1400 条折行长行）：

```
修复前：scrollHeight=111,098 / scrollTop=110,548（已到底）
修复后：scrollHeight=159,143 / scrollTop=158,593
低估：  48,045px ≈ 2184 行 × 22px（30%）
首屏：  526ms → 425ms（去掉估算反而更快）
```

### 修复

移除 `.log-row` 的 `content-visibility` / `contain-intrinsic-size`，
`scrollHeight` 恢复真实值，滚动条拖到底就是真底。

### 验证

```
滚到底：  lastLine=7420, visibleLast=7420, hiddenBelow=0
手动拖条：先回顶再设 scrollTop=max，等 1s → 末行完整可见（bottom 664 < body 672）
vitest     # 290 PASS（19 files）
vue-tsc    # 0 错
pnpm build # [check-version] ok: 0.7.4，ui/index.html 245.46 kB
```

截图：`docs/screenshots/console-bottom-after.png`（修复前状态代码已变更无法复现，此处只保留修复后）。
预览新增 `?long=1`（长行密集）用于回归这个场景。

### 打包（0.7.4）

- `dist/io.github.nixing178.jenkins-0.7.4-windows-x64.dbxp`，4,961,385 B，unsigned review candidate。
- sha256 `6e91ff5c4749105f…`；包内 6 条目，`ui/index.html` 247,813 B 与本地逐字节一致。
- sidecar `[verify-sidecar] ok: io.github.nixing178.jenkins/0.7.4`。

## 0.7.5 · 停止构建 / 取消排队的二次确认（2026-10-23）

### 现象

点击「停止」既不弹确认也不执行。`stopBuild()` 里的原生 `window.confirm()` 在宿主
webview 子 frame 中被静默忽略并返回 `false`，`if (!confirm) return` 直接吞掉操作。
`cancelQueueItem()` 同样写法，同一隐患。

### 方案

新增应用内确认弹窗 `ConfirmAction.vue`，`src/lib/confirmRequest.ts` 负责构造与校验
待确认目标；`stopBuild` / `cancelQueueItem` 只写 `pendingConfirm`，实际调用移入
`confirmPending`。详见 `docs/plan-stop-confirm.md`。

### 红 → 绿

```
新建 src/lib/confirmRequest.test.ts
vitest run src/lib/confirmRequest.test.ts
  红：Failed to load url ./confirmRequest — Test Files 1 failed, 0 test
  绿：✓ 9 tests passed
```

### 回归

```
go test ./...   # internal/jenkins ok 1.455s；internal/rpc ok 1.045s
vitest run      # 20 files / 299 PASS
vue-tsc         # 0 错
pnpm build      # [check-version] ok: 0.7.5，ui/index.html 249.45 kB
```

### 打包（0.7.5）

- `dist/io.github.nixing178.jenkins-0.7.5-windows-x64.dbxp`，4,962,300 B，unsigned review candidate。
- sha256 `6866817a3ec25b0b…`；包内 6 条目，`ui/index.html` 251,894 B 与本地逐字节一致。
- sidecar `[verify-sidecar] ok: io.github.nixing178.jenkins/0.7.5`（从 .dbxp 取出 `bin/windows-x64/dbx-plugin-jenkins.exe`，8,579,072 B）。

