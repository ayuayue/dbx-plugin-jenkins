import { createApp, h } from "vue";
import JobView from "./components/JobView.vue";
import { locale } from "./i18n";
import type { BuildSummary, JobDetail } from "./types";
import "./style.css";

/**
 * 构建列表「进度」列预览：只用于开发时肉眼检查列宽、配色与不确定态动画，
 * 用 `pnpm dev` 打开 /progress-preview.html；`?theme=dark` 切深色。
 */
locale.value = "zh";
const theme = new URLSearchParams(location.search).get("theme");
if (theme === "dark") document.documentElement.dataset.theme = "dark";

const total = 20;

function build(
  number: number,
  status: BuildSummary["status"],
  duration: number,
  estimatedDuration: number,
  cause: string,
): BuildSummary {
  return {
    number,
    result: status === "success" ? "SUCCESS" : status === "failure" ? "FAILURE" : "",
    building: status === "building",
    timestamp: Date.now() - (total - number + 1) * 3_600_000,
    duration,
    estimatedDuration,
    displayName: `#${number}`,
    causes: [{ shortDescription: cause }],
    status,
  };
}

const builds: BuildSummary[] = [
  build(632, "building", 38_000, 60_000, "Started by user 廖思源"),
  build(631, "success", 47_000, 52_000, "Started by user 廖思源"),
  build(630, "building", 9_000, 0, "Started by user 廖思源"),
  build(629, "success", 103_000, 100_000, "Started by user 廖思源"),
  build(628, "failure", 4_000, 45_000, "Started by user 陶凯"),
  build(627, "success", 51_000, 55_000, "Started by user 廖思源"),
];

const last = builds[0];

// #632 的时长估算约 63%，阶段步进算出 37%（1 完成 + 1 进行中 / 4）——
// 截图显示 37% 即证明阶段优先。
const stageSnapshot = {
  number: 632,
  stages: [
    { id: "1", name: "准备", status: "success", startTimeMillis: 0, durationMillis: 12_000, pauseDurationMillis: 0 },
    { id: "2", name: "编译", status: "in_progress", startTimeMillis: 0, durationMillis: 0, pauseDurationMillis: 0 },
    { id: "3", name: "单测", status: "not_executed", startTimeMillis: 0, durationMillis: 0, pauseDurationMillis: 0 },
    { id: "4", name: "部署", status: "not_executed", startTimeMillis: 0, durationMillis: 0, pauseDurationMillis: 0 },
  ],
};

const job: JobDetail = {
  name: "anjun-materials-web-admin",
  fullName: "Builds/anjun-materials-web-admin",
  description: "构建列表进度列预览",
  url: "https://jenkins.example/job/Builds/job/anjun-materials-web-admin/",
  color: "blue_anime",
  class: "WorkflowJob",
  buildable: true,
  inQueue: false,
  lastBuild: {
    number: last.number,
    result: String(last.result ?? ""),
    timestamp: last.timestamp,
    duration: last.duration,
    building: true,
  },
  lastSuccessfulBuild: null,
  lastFailedBuild: null,
  parameters: null,
  status: "building",
};

const refresh = {
  mode: "builds",
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
          h(JobView, {
            job,
            builds,
            loadingBuilds: false,
            readOnly: false,
            isPipeline: true,
            locale: "zh",
            refresh,
            changedBuilds: [],
            stageSnapshot,
          }),
        ],
      );
  },
};

createApp(Harness).mount("#app");
