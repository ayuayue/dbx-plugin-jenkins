import { createApp, h } from "vue";
import OverviewView from "./components/OverviewView.vue";
import { locale } from "./i18n";
import type { ExecutorInfo, Overview } from "./types";
import "./style.css";

/**
 * 概览页「构建执行状态」面板预览：用 `pnpm dev` 打开 /executor-preview.html，
 * `?theme=dark` 切深色。覆盖空闲 / 等待启动 / 构建中 / 收尾中 / 离线五种形态。
 */
locale.value = "zh";
const theme = new URLSearchParams(location.search).get("theme");
if (theme === "dark") document.documentElement.dataset.theme = "dark";

const overview: Overview = {
  version: "2.586",
  user: "demo-user",
  mode: "NORMAL",
  nodeDescription: "master",
  quietingDown: false,
  queueLength: 0,
  nodesOnline: 2,
  nodesTotal: 3,
  executorsBusy: 4,
  executorsTotal: 6,
};

const now = Date.now();

function executor(overrides: Partial<ExecutorInfo>): ExecutorInfo {
  return {
    node: "",
    nodeOffline: false,
    index: 1,
    idle: true,
    jobName: "",
    buildNumber: 0,
    building: false,
    timestamp: now,
    estimatedDuration: 0,
    duration: 0,
    result: "",
    pending: false,
    ...overrides,
  };
}

const executors: ExecutorInfo[] = [
  executor({
    node: "",
    index: 1,
    idle: false,
    building: true,
    jobName: "Builds/demo-materials-web",
    buildNumber: 632,
    estimatedDuration: 60_000,
    duration: 37_000,
  }),
  executor({ node: "", index: 2, idle: false, pending: true }),
  executor({ node: "", index: 3 }),
  executor({
    node: "agent-linux-01",
    index: 1,
    idle: false,
    building: true,
    jobName: "Ops/release-hotfix",
    buildNumber: 7,
    duration: 8_000,
  }),
  executor({
    node: "agent-linux-01",
    index: 2,
    idle: false,
    jobName: "Ops/nightly-cleanup",
    buildNumber: 118,
    result: "SUCCESS",
  }),
  executor({ node: "agent-win-02", nodeOffline: true, index: 1 }),
];

/** 概览页活跃档：有槽位被占用时的徽章与按钮三态。 */
const refresh = {
  mode: "overview",
  paused: false,
  inFlight: false,
  lastAt: now,
  intervalMs: 5000,
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
          h(OverviewView, {
            overview,
            loading: false,
            executors,
            loadingExecutors: false,
            refresh,
            onRefresh: () => {},
            onOpenExecutor: () => {},
          }),
        ],
      );
  },
};

createApp(Harness).mount("#app");
