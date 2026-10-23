import { createApp, h } from "vue";
import BuildView from "./components/BuildView.vue";
import { locale } from "./i18n";
import type { BuildDetail, ChangeSetItem } from "./types";
import "./style.css";

/**
 * 构建详情页「变更记录」预览：只用于开发时肉眼检查列宽、折叠、正文展开与配色。
 * `pnpm dev` 打开 /changes-preview.html；`?theme=dark` 切深色。
 */
locale.value = "zh";
const theme = new URLSearchParams(location.search).get("theme");
if (theme === "dark") document.documentElement.dataset.theme = "dark";

const now = Date.now();
const MINUTE = 60_000;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;

// 故意打乱顺序：最新的一条放在数组末尾，用来验证前端「最新在前」的排序。
const changes: ChangeSetItem[] = [
  {
    commitId: "a17c9e4b23f84d0e9b6a5c1f7d2e8b40c6a9f312",
    message: "chore(deps): 升级 pnpm 到 9.x",
    author: "开发者 A",
    timestamp: now - 6 * HOUR,
    fileCount: 2,
  },
  {
    commitId: "c3f8b1a97d6e42058f7c0b9a4e2d1f6b8c5a7d93",
    message: "fix(form): 修正切换选项后联动字段未回填",
    author: "开发者 B",
    timestamp: now - 3 * HOUR,
    fileCount: 4,
    body: "- 监听选项变化时同步重置联动数据\n- 新增 3 条表单用例\n- 回归：单选场景不受影响",
  },
  {
    commitId: "e5d2a8c47b9f6301d4e7c2a5b8f1d9e6a3c0b7f2",
    message: "docs: 补充 changelog 变更集字段说明",
    author: "开发者 C",
    timestamp: now - 2 * DAY,
    fileCount: 1,
  },
  {
    commitId: "b8f4d1e6a3c09b7250e8d5c2a7f4b1e9d6c3a0f5",
    message: "refactor(web): 抽出 table 的列宽计算",
    author: "开发者 B",
    timestamp: now - 3 * DAY,
    fileCount: 9,
    body: "- useTable 增加 totalWidth 计算\n- 保持对外 props 不变",
  },
  {
    commitId: "f2a7c4e9b6d31805a2f9e6c3b0d7a4f1e8c5b2d9",
    message: "test: 补 executors 四态单测",
    author: "开发者 A",
    timestamp: now - 4 * DAY,
    fileCount: 1,
  },
  {
    commitId: "d9b6e3f0a7c41d2859b6e3f0a7c41d2859b6e3f0",
    message: "feat(export): 导出列表新增流水号列",
    author: "开发者 C",
    timestamp: now - 5 * DAY,
    fileCount: 27,
    body: "- 导出统一加 serial_number\n- i18n：zh-CN / en / ja 三语",
  },
  {
    commitId: "7c1e8a5b2d9f6c3047e1a8b5d2f9c6a3e0b7d4f1",
    message: "fix(ci): Jenkins 构建产物路径写错",
    author: "开发者 A",
    timestamp: now - 6 * DAY,
    fileCount: 1,
  },
  {
    commitId: "3e9b6c2d8f5a1e7403b9c6d2f8a5e1b7c4d0a6f3",
    message: "feat(batch): 列表支持按条件批量补录",
    author: "开发者 C",
    timestamp: now - 7 * DAY,
    fileCount: 41,
    body: "- 新增批量补录入口\n- 校验关联关系的归属\n- 冲突时返回已存在单据号",
  },
  {
    commitId: "8a5d2f9c6b3e0a7418a5d2f9c6b3e0a7418a5d2f9",
    message: "style: 统一表格空态文案",
    author: "开发者 B",
    timestamp: now - 8 * DAY,
    fileCount: 3,
  },
  {
    commitId: "2f6c9b3e8a1d47052f6c9b3e8a1d47052f6c9b3e",
    message: "perf(batch): 批量并发参数按监控数据校准",
    author: "开发者 C",
    timestamp: now - 9 * DAY,
    fileCount: 2,
  },
  {
    commitId: "5b8e2f6a9c3d14705b8e2f6a9c3d14705b8e2f6a",
    message: "chore: 清理临时探针脚本",
    author: "开发者 A",
    timestamp: now - 10 * DAY,
    fileCount: 1,
  },
  {
    commitId: "9d3a7c1f5b8e24609d3a7c1f5b8e24609d3a7c1f",
    message: "release: 0.5.1 执行器跳转修复",
    author: "开发者 C",
    timestamp: now - 380 * DAY,
    fileCount: 6,
  },
];

const build: BuildDetail = {
  number: 128,
  result: "SUCCESS",
  building: false,
  timestamp: now - 26 * MINUTE,
  duration: 214_000,
  estimatedDuration: 220_000,
  displayName: "#128",
  fullDisplayName: "Builds » demo-schedule-web #128",
  url: "https://jenkins.example/job/Builds/job/demo-schedule-web/128/",
  causes: [{ shortDescription: "Started by user 开发者 C" }],
  status: "success",
  changeSet: changes,
  artifacts: [],
};

const refresh = {
  mode: "build",
  paused: false,
  inFlight: false,
  lastAt: Date.now(),
  intervalMs: 3000,
  refreshing: false,
  refreshed: false,
};

const Harness = {
  setup() {
    return () =>
      h(
        "div",
        { style: "height:100vh;display:flex;background:var(--bg);font-family:var(--font)" },
        [
          h(BuildView, {
            build,
            jobFullName: "Builds/demo-schedule-web",
            readOnly: false,
            locale: "zh",
            refresh,
          }),
        ],
      );
  },
};

createApp(Harness).mount("#app");
