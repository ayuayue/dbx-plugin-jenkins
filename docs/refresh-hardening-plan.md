# 自动刷新治理计划（v0.4.3）

> 依据：2026-10-22 全量排查结论。共 10 项问题，按 P1 → P3 分 6 步落地。
> 纪律：先写失败测试跑红 → 写实现跑绿 → 记录 `docs/test-evidence.md`。

## 一、问题 → 步骤映射

| 编号 | 问题 | 级别 | 步骤 |
|---|---|---|---|
| R1 | 构build 详情页无任何自动刷新，且无刷新按钮 | P1 | Step 2 |
| R2 | 队列页无自动刷新 | P1 | Step 3 |
| R3 | 串行链被 RPC 超时拖死（stages 120s / 其它 30s） | P1 | Step 1 |
| R4 | 无 `visibilitychange`，回前台不立即刷新；后台空转 | P2 | Step 1 |
| R5 | 轮询静默吞错，数据陈旧无提示 | P2 | Step 1 |
| R6 | `pendingBuild` expired 后不补刷列表 | P2 | Step 4 |
| R7 | 空闲时左侧任务列表不刷新 | P2 | Step 6（可选，D4） |
| R8 | ConsoleView 无视后台、`text` 无上限 | P3 | Step 5 |
| R9 | `stopBuild` 的 800ms 延迟刷新无目标校验 | P3 | Step 4 |
| R10 | 手动刷新与轮询的并发窗口（restart 时序 + token） | P3 | Step 4 |

## 二、步骤与改动

### Step 1｜轮询基础设施（R3 / R4 / R5）

- `src/lib/refresh.ts`
  - `AutoRefresherOptions` 新增 `maxConsecutiveFailures?: number`（0 表示不限）与 `onPaused?: (reason: "failures") => void`。
  - tick 内维护 `failures`：task 抛错 `+1`，成功清零；达到阈值时 `running = false` 并回调 `onPaused`。
  - `AutoRefresher` 新增 `trigger(): void`：立即执行一轮并重置倒计时（供回前台使用）。
  - 新增常量 `POLL_TIMEOUT_MS = 15_000`。
- `src/api.ts`：`job` / `builds` / `stages` 增加可选 `timeoutMs` 参数，轮询传 `POLL_TIMEOUT_MS`，手动刷新保持原值（stages 仍 120s）。
- `src/App.vue`
  - 新增 `refreshPaused = ref(false)` 与顶部提示条（与错误 banner 并列，不占用 `errorMessage`）。
  - `startAutoRefresh` 传 `maxConsecutiveFailures: 3`，`onPaused` 置 `refreshPaused = true`。
  - 手动刷新（`refreshCurrent` 各分支）时清 `refreshPaused`。
  - 注册 `visibilitychange`：隐藏时 `stopAutoRefresh()`；可见时按当前 `autoRefreshMode` 重建并 `trigger()` 立即刷一轮。组件卸载时移除监听。

### Step 2｜构建详情页轮询（R1）

- `refresh.ts` 新增纯函数 `shouldPollBuildDetail(build: BuildDetail | null): boolean`（`building === true || status === "building"`）。
- `App.vue`：`autoRefreshMode` 增加 `"build"`，条件 `view === "build" && shouldPollBuildDetail(selectedBuild)`。
- 新增 `pollBuildDetail()`：静默拉 `jenkinsApi.build(job, number)`，校验 `view`/`number`/`selectToken` 未变后写 `selectedBuild`。
- `BuildView.vue`：工具栏加刷新按钮，emit `refresh`；`App.vue` 绑 `refreshBuild`。

### Step 3｜队列页轮询（R2）

- `refresh.ts` 新增常量 `QUEUE_AUTO_REFRESH_INTERVAL_MS = 5000`。
- `autoRefreshMode` 增加 `"queue"`（条件见决策 D2）。
- 新增 `pollQueue()`：静默拉 `jenkinsApi.queue()`，不动 `loadingView`，失败静默。

### Step 4｜延迟刷新收敛（R6 / R9 / R10）

- `settlePendingBuild`：`expired` 时同样 `loadJobs(folder, true)` 一次。
- `stopBuild` 的 800ms 回调：记录 `job` / `number` / `view` 快照，回调内校验一致后才刷新。
- `refreshJob` / `refreshStages`：`restart()` 提前到 `await` 之前，避免请求窗口内 tick 再发起一轮。
- `pollBuilds`：取 `selectToken` 快照，响应后校验，防止覆盖手动刷新结果。

### Step 5｜控制台轮询（R8）

- `src/lib/console.ts` 新增 `trimLogTail(text: string, maxChars: number): string`，按行边界保留尾部。
- `ConsoleView.vue`：`text` 累积后调用 `trimLogTail(text, MAX_TEXT_CHARS)`（2MB）；`document.hidden` 时不排下一轮，改为监听 `visibilitychange` 恢复后续拉。

### Step 6｜空闲任务列表（R7，可选，见 D4）

- 若启用：`autoRefreshMode` 增加 `"jobs"` 态（`view === "jobs"` 且无构建中），间隔 30s，只刷 `loadJobs(silent)`。

## 三、决策点

| 编号 | 议题 | 选项 | 推荐 |
|---|---|---|---|
| D1 | 构建详情页刷新方式 | A 3s 轮询 + 手动按钮，结束自动停；B 只加按钮；C 只轮询 | **A** |
| D2 | 队列页轮询条件 | A 固定 5s；B 仅队列非空时轮询；C 3s | **A** |
| D3 | 失败护栏 | A 连续 3 次失败即停 + 顶部提示；B 降频重试不停；C 只提示不停 | **A** |
| D4 | 空闲任务列表 | A 不加（保持现状）；B 30s 轻量轮询；C 10s | **A** |
| D5 | 控制台日志内存 | A 保留尾部 2MB + 后台暂停；B 只做后台暂停；C 保持现状 | **A** |

无需决策、按推荐直接做：轮询专用 15s 超时、expired 补刷、`stopBuild` 守卫、`restart` 时序与 token 校验。

## 四、测试计划（TDD，AI 代跑）

`src/lib/refresh.test.ts` 新增：

1. `createAutoRefresher`：连续失败达 `maxConsecutiveFailures` 后自动停止，不再排下一轮。
2. `createAutoRefresher`：中间成功一次即把失败计数清零。
3. `createAutoRefresher.trigger()`：立即执行一轮并重置倒计时。
4. `shouldPollBuildDetail`：`building` 真值、`status=building`、空值、已结束四种判定。

`src/lib/console.test.ts` 新增：

5. `trimLogTail`：未超限原样返回；超限保留尾部且不切断末行；空串安全。

## 五、验收

- `pnpm test`（vitest）全绿。
- `pnpm build` 通过（含 `check-version.mjs`）。
- 版本号四处同步：`manifest.json` / `package.json` / `backend/main.go` / `backend/internal/jenkins/client.go`（0.4.3）。
- `docs/test-evidence.md` 追加红/绿记录。
