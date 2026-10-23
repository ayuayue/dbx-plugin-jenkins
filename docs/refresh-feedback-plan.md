# 自动刷新提示与页面反馈优化（0.4.4）

> 现状：`autoRefreshing` 只是「轮询已启用」的布尔值，渲染成一个静止绿点。
> 用户看不出「这一轮请求有没有跑」「上次成功是什么时候」「点了刷新有没有反应」。

## 一、要解决的问题

| # | 现状 | 期望 |
|---|---|---|
| F1 | 绿点静止，不区分「请求中 / 等待下一轮」 | 圆点有实心/脉冲与呼吸之分 |
| F2 | 没有「上次更新」时间，无法判断数据是否陈旧 | 显示 `HH:mm:ss` |
| F3 | 暂停时只有顶部横幅，视线在面板里时看不到 | 圆点就地变琥珀 + 文案「已暂停」 |
| F4 | 点「刷新」到数据回来之间没有任何反馈，可重复点击 | 按钮三态：刷新 → 刷新中… → 已更新 |
| F5 | 新构建出现、构建状态翻转时列表静默变化 | 变化行左侧短暂高亮 |
| F6 | 队列页有常驻轮询但完全没有指示 | 与其它页统一 |

## 二、实现骨架

1. `src/lib/refresh.ts`：`AutoRefresherOptions` 新增 `onRunStart?: () => void`、`onRunEnd?: (ok: boolean) => void`。
   - 前提：`pollBuilds` / `pollBuildDetail` / `pollStages` / `pollQueue` 去掉内部 `catch`，错误冒泡给 refresher 统一计数（否则 `onRunEnd` 永远拿到成功）。
2. `src/lib/refreshStatus.ts`（新增，纯函数，可单测）：
   - `refreshBadge(mode, paused, inFlight)` → `{ tone: "active" | "paused" | "idle", labelKey }`；
   - `formatClockTime(timestamp)` → `HH:mm:ss`；
   - `diffBuildRows(prev, next)` → 变化的构建号集合（供行高亮）。
3. `src/components/RefreshBadge.vue`（新增）：四处复用的徽章，props `mode / inFlight / paused / lastAt / intervalMs`，统一维护样式与 tooltip。
4. `src/App.vue`：新增 `pollInFlight`、`lastRefreshAt` ref；`onRunStart`/`onRunEnd` 接线；`loadJobs` / `selectJob` / `loadStages` / `loadQueue` 成功分支调用 `markRefreshed()`（手动刷新也算「上次更新」）。
5. 组件接入：`JobView` / `StageView` / `BuildView` / `QueueView` 用 `RefreshBadge` 替换各自重复的 `.auto-refresh` 样式；`QueueView` 新增徽章位。
6. 按钮三态：`refreshCurrent` 外层包 `manualRefreshing` / `manualJustDone` 两个 ref，按钮 `:disabled` + 文案切换，1.5s 后复原。

## 三、决策点

| 编号 | 议题 | 选项 | 推荐 |
|---|---|---|---|
| D1 | 刷新指示形态 | A 圆点 + 「自动刷新中 · 12:31:05」，请求中实心脉冲/空闲呼吸，tooltip 写间隔与护栏规则；B 只加时间不加请求态；C 改顶部细进度条；D 保持现状 | **A** |
| D2 | 暂停态呈现 | A 圆点就地变琥珀「已暂停 · 连续 3 次失败」+ 保留横幅；B 只留横幅；C 去掉横幅只留圆点 | **A** |
| D3 | 手动刷新反馈 | A 三态（刷新 → 刷新中… → 已更新 1.5s）；B 只做禁用 + 刷新中…；C 不改 | **A** |
| D4 | 数据变化反馈 | A 新增行左侧 1.5s 绿条 + 状态翻转的 pill 轻闪；B 只做行高亮；C 不做 | **A** |
| D5 | 覆盖范围 | A 构建列表 / 构建详情 / 阶段视图 / 队列页四处统一；B 只改前三个 | **A** |

## 四、测试计划（TDD，AI 代跑）

`src/lib/refresh.test.ts` 追加：

1. `onRunStart` 在 task 前触发、`onRunEnd(true)` 在成功后触发。
2. task 抛错时 `onRunEnd(false)` 触发，且仍按护栏计数。

`src/lib/refreshStatus.test.ts`（新增）：

3. `refreshBadge`：暂停优先于请求中；请求中与空闲的 tone 区分；无 mode 时 idle。
4. `formatClockTime`：补零、跨小时、null 返回空串。
5. `diffBuildRows`：新增行、状态翻转、无变化返回空集。

## 五、验收

- `pnpm test` 全绿；`pnpm build` 通过。
- 版本号四处同步升 `0.4.4`（`manifest.json` / `package.json` / `backend/main.go` / `client.go`）。
- `docs/test-evidence.md` 追加红/绿记录；`README.md` 自动刷新章节补「页面反馈」说明。

## 六、实施结果（2026-10-22，0.4.4）

- 新增 `src/lib/refreshStatus.ts`（`refreshBadge` / `formatClockTime` / `diffBuildRows` / `RefreshUi`）与 `src/components/RefreshBadge.vue`（四态徽章，含 `prefers-reduced-motion` 降级）。
- `AutoRefresher` 增加 `onRunStart` / `onRunEnd(ok)`；四个 `pollXxx` 去掉内部 `catch`，错误冒泡给护栏——**护栏的连续失败计数此前实际无法触发**，本次一并修正。
- `App.vue` 新增 `pollInFlight` / `lastRefreshAt` / `manualRefreshing` / `manualJustDone` / `changedBuilds`，并用 `refreshUi` 聚合对象传给各视图。
- 四个视图（JobView / StageView / BuildView / QueueView）统一接 `refresh: RefreshUi`：徽章 + 按钮三态；JobView 额外接 `changedBuilds` 做行高亮。
- 测试：红→绿两轮，`14 files / 201 tests` 全绿；`pnpm build` → `ui/index.html` 221.19 kB。
- 目视：`docs/refresh-badge-preview.png`（浅色）、`docs/refresh-badge-preview-dark.png`（深色）。
- 产物：`dist/io.github.nixing178.jenkins-0.4.4-windows-x64.dbxp`（4,945,103 bytes，sha256 `dc841278…`），包内 UI 与本地逐字节一致，sidecar identity `0.4.4`。
