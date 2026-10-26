# 页面前进 / 后退（鼠标侧键）支持（0.10.0）

> 需求：鼠标的「后退 / 前进」侧键（XButton1 / XButton2）能控制插件内的页面导航。

## 一、现状与问题

插件 UI 是单文件 SPA，所有「页面」都是 `App.vue` 里的一个 ref：

| 导航状态 | 变量 |
|---|---|
| 顶层页面 | `view: overview \| jobs \| build \| stages \| queue \| nodes` |
| 当前视图 tab | `activeView` |
| 文件夹层级 | `folder` |
| 选中任务 | `selectedJob.fullName` |
| 构建详情 | `selectedBuild.number` |

**这些状态一个都没进浏览器 history**。所以即使宿主把侧键当作浏览器后退，也只会退宿主自己（甚至退出工作台），插件画面纹丝不动。

## 二、三条必须先讲清的技术事实

| # | 事实 | 后果 |
|---|---|---|
| T1 | Chromium 在 Windows 上把侧键转成 `browser-backward` / `browser-forward` 命令；**是否同时派发 DOM `mouseup`（`button===3/4`）取决于宿主处理方式** | 只做一个通道必然有用户按了没反应 |
| T2 | 插件被宿主以 iframe / webview 加载（单文件 `ui/index.html`）；iframe 的 `pushState` 会进入 tab 的联合会话历史 | 只要 push 了，宿主执行的 back/forward 会落到 iframe 的 `popstate`，我们能接住 |
| T3 | `history.pushState` 在 `file://`（origin=null）和 `about:srcdoc` 下会抛 `SecurityError` | 必须运行时探测，不可用就降级到纯 DOM 通道 |

结论：**双通道 + 去重**是唯一稳妥路线。

## 三、方案

新增三个纯逻辑模块（可单测），`App.vue` 只做接线：

```
src/lib/navStack.ts      导航栈（唯一真相）：条目、游标、push/back/forward/canX
src/lib/mouseNav.ts      侧键解析 + 双通道去抖器
src/lib/historyBridge.ts pushState 能力探测与写入（file:// 降级）
```

### 数据流

```
用户导航动作 ──► navStack.push(entry) ──► historyBridge.push(index)   [镜像浏览器栈]
                                                     │
鼠标侧键 ─┬─ DOM mouseup(button 3/4) ──► dedupe ─────┤
          └─ 宿主 back ──► popstate ──► dedupe ─────┴─► navStack.back()/forward()
                                                          │
                                                     applyEntry(entry) 回放
```

- **浏览器栈与内部栈 1:1 镜像**，`pushState` 的 `state` 里存 `index`；`popstate` 只读 `index` 校正游标，不信任其它字段。不用「哨兵 + 回补」那套 hack。
- **去抖**：一次侧键可能同时产生 DOM 事件与 `popstate`，用 120ms 窗口按方向去重，只执行一次。
- **file:// 降级**：`pushState` 抛错 → `available=false`，只走 DOM 通道（此时宿主若已吞事件则无法生效，见风险 R1）。

### 导航条目

```ts
interface NavEntry {
  view: AppView;
  activeView: string;   // 视图 tab
  folder: string;       // 文件夹路径
  job: string;          // selectedJob.fullName，空串表示无选中
  build: number | null; // 仅 view==="build"
}
```

### 哪些动作入栈

| 用户动作 | 当前入口 | 入栈 | 回放 |
|---|---|---|---|
| 侧栏切页 | `switchView` | ✅ | `refreshOnEnter()` |
| 切视图 tab | `activateView` | ✅ | 设 `activeView` → `loadJobs("")` |
| 进文件夹 / 面包屑上跳 | `loadJobs(path)` | ✅ | `loadJobs(entry.folder, true)` |
| 点任务 / 执行器行跳转 | `selectJob` | ✅ | `loadJobs` → `selectJob`（静默） |
| 打开构建详情 | `openBuild` | ✅（含构建号） | `selectJob` → `openBuild(n, false)` |
| 打开阶段视图 | `openStages` | ✅ | `selectJob` → `loadStages` |
| 页面内「返回」 | `closeBuild` / `backToJobs` | ❌ | 改为 `navBack()`，与侧键后退同义 |
| 弹窗 / 编辑器（参数、任务编辑、批量、确认） | 各 `ref` | ❌ | 见 D6 |

### UI 反馈

`←` / `→` 两个箭头按钮放 topbar 左侧 brand 右侧，`:disabled` 反映 `canBack` / `canForward`，tooltip 写「后退（鼠标侧键 / 返回按钮）」。既让功能可发现，也方便无侧键的触摸板验收。

## 四、决策点

| 编号 | 议题 | 选项 | 推荐 |
|---|---|---|---|
| D1 | 历史粒度 | A 全量（切页/视图/文件夹/任务/构建/阶段）<br>B 仅切页 + 构建/阶段（不含任务与文件夹）<br>C 仅四个侧栏页 | **A** |
| D2 | 页面内「返回」按钮语义 | A 改为「后退一步」（与侧键一致，前进能回到刚才的构建）<br>B 保持现状直接切页，仅侧键用栈 | **A** |
| D3 | 是否加 `←/→` 箭头按钮 | A 加（可发现 + 触摸板可用）<br>B 不加，纯侧键 | **A** |
| D4 | 是否支持 `Alt+←/→` | A 支持<br>B 不做（与宿主快捷键易冲突） | **B** |
| D5 | 历史栈深度 | 20 / **50** / 100 | **50** |
| D6 | 有弹窗时按后退 | A 先关弹窗，不消耗历史（前进不恢复弹窗）<br>B 忽略弹窗直接退页面 | **A** |
| D7 | 版本号 | 0.9.2 → **0.10.0**（新功能） | **0.10.0** |

## 五、TDD 测试计划（红 → 绿，AI 代跑 vitest）

`src/lib/navStack.test.ts`
1. push 后 `canBack=false`、再 push 置 `canBack=true`；新 push 截断 forward 分支
2. 连续 push 相同条目只留一条（避免抖两次多点一次）
3. 超过上限丢最旧，游标同步前移
4. `back()` 到底返回 `false` 且游标不动；`forward()` 到顶同理
5. `restore(index)` 越界时夹紧

`src/lib/mouseNav.test.ts`
6. `button 3 → "back"`、`4 → "forward"`、`0/1/2 → null`
7. 去抖器：120ms 内同通道重复只放行一次；先 back 后 forward 不受影响
8. 窗口过期后再次放行

`src/lib/historyBridge.test.ts`
9. `pushState` 抛错 → `available=false`，`push()` 返回 `false` 不抛
10. 正常时 `push()` 写入 `index` 并返回 `true`

## 六、实施步骤

1. **第 0 步（诊断，先做）**：临时在 `App.vue` 的 `mouseup` / `popstate` 上加 `console.info`，用真机侧键确认宿主走哪条通道（T1）。只加日志，不改行为，验证完即成正式实现的埋点。
2. 红：三个模块的测试全部失败 → 源哥确认红图。
3. 绿：实现三个模块 → 单测全绿。
4. `App.vue` 接线：`navigate()` 统一入口替换现有导航调用点；`navBack/navForward/applyEntry`；`onMounted` 注册监听、`onBeforeUnmount` 清理。
5. 预览页人工验收（浅色 + 深色），截图归档 `docs/nav-history-preview.png`。
6. 版本号四处同步 0.10.0 → `pnpm build` → 打包校验 → 写 `docs/test-evidence.md`。

## 七、风险与兜底

| 编号 | 风险 | 兜底 |
|---|---|---|
| R1 | 宿主吞掉侧键且自身不做导航（T1 的第三种情况） | 无解，需 DBX 宿主侧支持；第 0 步就能发现，届时上报而不是硬扛 |
| R2 | `file://` 下 pushState 不可用 | 自动降级 DOM 单通道；顶部箭头按钮始终可用 |
| R3 | pushState 污染宿主历史：退出插件后宿主后退可能回到插件页 | 可接受（标准 SPA 行为）；不额外 push 哨兵，避免多一条垃圾记录 |
| R4 | 回放与自动轮询 / 进入刷新撞车 | 回放统一走 `applyEntry`，复用现有 token 机制；浏览器前进后退期间 `enterRefreshBusy` 不参与 |
| R5 | 拖拽选择、输入框里的侧键被误判 | 只认 `button 3/4`，不与左/中/右键重叠；侧键不影响输入 |

## 八、验收

- `pnpm test` 全绿（新增 3 个测试文件），`node node_modules/vue-tsc/bin/vue-tsc.js --noEmit` 无新错误。
- 真机链路：切页 → 点任务 → 进文件夹 → 开构建 → 侧键后退四次逐层回退 → 侧键前进四次逐层复原。
- 有弹窗时按后退只关弹窗（D6 若选 A）。
- `pnpm build` → `npx @dbx-app/plugin-cli@0.1.9 package .` → 包内 `ui` 与本地逐字节一致、sidecar 版本 0.10.0。

## 九、实施记录（2026-10-26，0.10.0）

已落地：

| 文件 | 改动 |
|---|---|
| `src/lib/navStack.ts` | 新增：导航栈（`push` 返回 `{index, added}`、`back`/`forward`/`restore` 夹紧、上限 50、相同条目合并） |
| `src/lib/mouseNav.ts` | 新增：`resolveMouseButton`（3/4）、`createNavDeduper`、`NAV_MOUSE_FALLBACK_MS=90`、`NAV_DEDUPE_MS=150` |
| `src/lib/historyBridge.ts` | 新增：`push`/`replace`/`go`/`index`，`pushState` 抛错即永久降级 |
| `src/App.vue` | `navEntry`/`recordNav`/`applyEntry`/`navBack`/`navForward`/`handleMouseNavButton`/`handlePopState`；`switchView`/`openJob`/`navigateFolder`/`chooseView`/`openBuild`/`openStages` 接入记录；`closeBuild`/`backToJobs` 改为后退；`loadJobs` 增加 `autoSelect`；topbar 新增 ←/→ |
| `src/i18n.ts` | `navBack` / `navForward`（中英） |
| `README.md` | 版本号 + 「页面前进 / 后退」章节 |
| 版本号 | manifest / package / main.go / client.go 四处同步 `0.10.0` |

验证结果（环境恢复后，2026-10-26 11:43–11:53）：

| 项 | 结果 |
|---|---|
| `vitest run` | 24 files / **373 passed**（新增 29 条全绿） |
| 变异验证 | 把 `button === 3` 改成 `2` → 2 条红；还原后全绿（替代缺失的红图） |
| `vue-tsc --noEmit` | exit 0 |
| `pnpm build` | `[check-version] ok: 0.10.0`，`ui/index.html` 281,691 B |
| `go test ./...` | jenkins / rpc 均 ok |
| 打包 | `io.github.nixing178.jenkins-0.10.0-windows-x64.dbxp` 4,988,865 B，sha256 `c1eb23de…`，包内 ui 逐字节一致，sidecar 含 `0.10.0` ×2 |

仍未验证：

- ~~真机侧键通道~~ → **2026-10-26 12:00 源哥实测通过**（安装 0.10.0 后侧键前进 / 后退正常）。
- 预览截图未出：Headless Chrome 在本机启动即退出（无输出）。
