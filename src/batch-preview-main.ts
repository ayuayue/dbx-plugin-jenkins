import { createApp, h, ref } from "vue";
import BatchBuildDialog from "./components/BatchBuildDialog.vue";
import JobList from "./components/JobList.vue";
import { syncLocale } from "./i18n";
import { batchTargets, rememberTargets } from "./lib/batchBuild";
import type {
  BatchBuildResult,
  BatchTargetInfo,
  JobSummary,
  ParameterDefinition,
  ViewInfo,
} from "./types";
import "./style.css";

/**
 * 批量构建预览页：`pnpm dev` 打开 /batch-preview.html，`?theme=dark` 切深色。
 * 只用于肉眼核对多选态与弹窗三态，不参与打包；数据全为 demo-*。
 */
const theme = new URLSearchParams(location.search).get("theme");
if (theme === "dark") document.documentElement.dataset.theme = "dark";
syncLocale("zh-CN");

function demoJob(
  name: string,
  status: JobSummary["status"],
  color: string,
  building = false,
): JobSummary {
  return {
    name,
    fullName: `release/${name}`,
    url: `http://ci.example.com/job/release/job/${name}/`,
    color,
    isFolder: false,
    status,
    building,
    lastBuild: { number: 42, result: "SUCCESS", timestamp: 0, duration: 0, building },
  };
}

const jobs = ref<JobSummary[]>([
  demoJob("demo-api", "success", "blue"),
  demoJob("demo-web", "building", "blue_anime", true),
  demoJob("demo-worker", "failure", "red"),
  demoJob("demo-gateway", "not_built", "notbuilt"),
]);

const views: ViewInfo[] = [
  { name: "all", description: "", url: "" },
  { name: "Release", description: "", url: "" },
];

const selected = ref(["release/demo-api", "release/demo-web", "release/demo-worker"]);
const phase = ref<"inspecting" | "confirm" | "running" | "done" | "hidden">("confirm");

/** 跨视图缓存的演示：切视图后仍能拿到名字与状态。 */
const targetMap = rememberTargets({}, jobs.value);

/** 演示用的共享参数并集（实际由后端 batchInspect 返回）。 */
const sharedParams: ParameterDefinition[] = [
  {
    name: "SKIP_CODE_CHECKOUT",
    type: "BooleanParameterDefinition",
    description: "是否跳过拉取代码过程（勾选跳过）",
    defaultValue: "false",
  },
  {
    name: "SKIP_BUILD",
    type: "BooleanParameterDefinition",
    description: "是否跳过编译过程（勾选跳过）",
    defaultValue: "false",
  },
];

function inspectTargets(): BatchTargetInfo[] {
  return batchTargets(selected.value, targetMap).map((target, index) => ({
    ...target,
    buildable: true,
    inQueue: false,
    parameters: index === 0 ? sharedParams : null,
  }));
}

const results: BatchBuildResult[] = [
  {
    job: "release/demo-api",
    success: true,
    skipped: false,
    queueId: 1201,
    reason: "",
    error: "",
  },
  {
    job: "release/demo-web",
    success: false,
    skipped: false,
    queueId: 0,
    reason: "forbidden",
    error: "forbidden: demo-user 缺少 Job/Build 权限 (HTTP 403)",
  },
  {
    job: "release/demo-worker",
    success: false,
    skipped: true,
    queueId: 0,
    reason: "in_queue",
    error: "",
  },
];

const Harness = {
  setup() {
    function toggleBatchJob(fullName: string): void {
      selected.value = selected.value.includes(fullName)
        ? selected.value.filter((name) => name !== fullName)
        : [...selected.value, fullName];
    }

    return () =>
      h("div", { style: "display:flex;height:100vh;background:var(--bg)" }, [
        h(JobList, {
          jobs: jobs.value,
          selected: "",
          folder: "",
          loading: false,
          views,
          activeView: "Release",
          readOnly: false,
          batchMode: true,
          batchSelected: selected.value,
          batchOutsideCount: 1,
          onToggleBatchJob: toggleBatchJob,
          onSelectAllVisible: () => {
            selected.value = jobs.value.map((job) => job.fullName);
          },
          onClearBatch: () => {
            selected.value = [];
          },
          onBuildBatch: () => {
            phase.value = "confirm";
          },
          onToggleBatchMode: () => undefined,
          onRefresh: () => undefined,
        }),
        h("div", { style: "flex:1;padding:24px;display:flex;flex-direction:column;gap:10px" }, [
          h("p", { style: "margin:0;color:var(--muted);font-size:12px" }, "预览控制（不参与打包）"),
          h("div", { style: "display:flex;gap:8px" }, [
            button("读取中", () => (phase.value = "inspecting")),
            button("确认态", () => (phase.value = "confirm")),
            button("执行中", () => (phase.value = "running")),
            button("结果态", () => (phase.value = "done")),
            button("关闭", () => (phase.value = "hidden")),
          ]),
          h(
            "p",
            { style: "margin:0;color:var(--muted);font-size:12px;line-height:1.7" },
            `已选 ${selected.value.length} 个任务；弹窗清单按列表顺序生成。`,
          ),
        ]),
        phase.value === "hidden"
          ? null
          : h(BatchBuildDialog, {
              targets: inspectTargets(),
              phase: phase.value,
              skipBusy: true,
              results: phase.value === "done" ? results : [],
              progress:
                phase.value === "running"
                  ? { done: 12, total: 30 }
                  : { done: 3, total: 3 },
              error: "",
              parameters: sharedParams,
              paramValues: { SKIP_CODE_CHECKOUT: true, SKIP_BUILD: false },
              onConfirm: () => {
                phase.value = "running";
                window.setTimeout(() => (phase.value = "done"), 1200);
              },
              onRetry: () => {
                phase.value = "running";
                window.setTimeout(() => (phase.value = "done"), 1200);
              },
              onRemove: (fullName: string) => {
                selected.value = selected.value.filter((name) => name !== fullName);
              },
              onClose: () => (phase.value = "hidden"),
            }),
      ]);

    function button(label: string, onClick: () => void) {
      return h(
        "button",
        {
          type: "button",
          class: "ghost",
          style:
            "border:1px solid var(--border);background:transparent;color:var(--text);border-radius:6px;padding:6px 10px;font-size:12px",
          onClick,
        },
        label,
      );
    }
  },
};

createApp(Harness).mount("#app");
