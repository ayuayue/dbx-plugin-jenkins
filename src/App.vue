<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import logoUrl from "../assets/plugin.svg";
import { describeError, jenkinsApi, setConnectionId } from "./api";
import BuildView from "./components/BuildView.vue";
import ConfirmDelete from "./components/ConfirmDelete.vue";
import JobEditor from "./components/JobEditor.vue";
import JobList from "./components/JobList.vue";
import JobView from "./components/JobView.vue";
import NodesView from "./components/NodesView.vue";
import OverviewView from "./components/OverviewView.vue";
import ParamDialog from "./components/ParamDialog.vue";
import QueueView from "./components/QueueView.vue";
import StageView from "./components/StageView.vue";
import ToastHost from "./components/ToastHost.vue";
import ViewEditor from "./components/ViewEditor.vue";
import { locale, syncLocale, t } from "./i18n";
import { errorText } from "./lib/errors";
import { createToastQueue, type ToastItem, type ToastTone } from "./lib/toast";
import {
  AUTO_REFRESH_INTERVAL_MS,
  STAGE_AUTO_REFRESH_INTERVAL_MS,
  createAutoRefresher,
  createPendingBuild,
  evaluatePendingBuild,
  hasRunningBuild,
  hasRunningStage,
  type AutoRefresher,
  type PendingBuild,
} from "./lib/refresh";
import {
  appendJobName,
  containsJob,
  isNamedView,
  needsJoinView,
  resolveJoinAction,
} from "./lib/viewRefresh";
import type {
  BuildDetail,
  BuildSummary,
  CreateJobPayload,
  CreateViewPayload,
  JobConfigResponse,
  JobDetail,
  JobSummary,
  JobUpdateFields,
  NodeInfo,
  Overview,
  QueueItem,
  StageMatrix,
  ViewConfigResponse,
  ViewInfo,
  ViewUpdateFields,
} from "./types";

interface Identity {
  name: string;
  version: string;
  mode: string;
}

const connectionName = ref("");
const identity = ref<Identity | null>(null);
const readOnly = ref(true);

const view = ref<"overview" | "jobs" | "build" | "stages" | "queue" | "nodes">("overview");
const folder = ref("");
const jobs = ref<JobSummary[]>([]);
const views = ref<ViewInfo[]>([]);
const activeView = ref("");
const selectedJob = ref<JobDetail | null>(null);
const builds = ref<BuildSummary[]>([]);
const selectedBuild = ref<BuildDetail | null>(null);
const stageMatrix = ref<StageMatrix | null>(null);
const loadingStages = ref(false);
const buildReturnView = ref<"jobs" | "stages">("jobs");
const overview = ref<Overview | null>(null);
const queueItems = ref<QueueItem[]>([]);
const nodes = ref<NodeInfo[]>([]);

const loadingJobs = ref(false);
const loadingJob = ref(false);
const loadingBuilds = ref(false);
const loadingView = ref(false);
const showParams = ref(false);
const errorMessage = ref("");
/** 操作提示改为浮层 toast：只占视觉，不挤压页面布局。 */
const toasts = ref<ToastItem[]>([]);
const toastQueue = createToastQueue({
  onChange: (items) => (toasts.value = items),
});

type JobEditorState =
  | { mode: "create" }
  | { mode: "edit"; config: JobConfigResponse };

type ViewEditorState =
  | { mode: "create" }
  | { mode: "edit"; view: ViewInfo; config: ViewConfigResponse };

interface DeleteTarget {
  kind: "job" | "view";
  name: string;
  fullName?: string;
}

const jobEditor = ref<JobEditorState | null>(null);
const viewEditor = ref<ViewEditorState | null>(null);
const deleteTarget = ref<DeleteTarget | null>(null);
const editorBusy = ref(false);
const viewJobPool = ref<string[]>([]);

const canManageView = computed(() => isNamedView(activeView.value));

/** 已有视图名，用于编辑视图时的重名提示（Jenkins 判重区分大小写）。 */
const viewNames = computed(() =>
  views.value.map((item) => item.name).filter((item) => item !== ""),
);

const connectionLabel = computed(() => {
  if (identity.value?.version) return `Jenkins ${identity.value.version}`;
  return connectionName.value || t("unknownConnection");
});

const isPipeline = computed(() =>
  (selectedJob.value?.class ?? "").toLowerCase().includes("workflowjob"),
);

/** 刚触发、还没在构建列表里出现的构建；存在时构建列表保持自动刷新。 */
const pendingBuild = ref<PendingBuild | null>(null);

/** 当前视图是否需要自动刷新：构建列表有构建中 / 刚触发构建，或阶段视图有进行中的构建/阶段。 */
const autoRefreshMode = computed<"builds" | "stages" | null>(() => {
  if (view.value === "jobs" && selectedJob.value) {
    if (hasRunningBuild(builds.value)) return "builds";
    if (pendingBuild.value?.job === selectedJob.value.fullName) return "builds";
  }
  if (view.value === "stages" && hasRunningStage(stageMatrix.value)) {
    return "stages";
  }
  return null;
});

const autoRefreshing = computed(() => autoRefreshMode.value !== null);

let autoRefresher: AutoRefresher | null = null;

let jobsToken = 0;
let selectToken = 0;
let stageToken = 0;

function setError(error: unknown): void {
  errorMessage.value = errorText(describeError(error), locale.value);
}

/** 成功提示走 success，纯提示（只读拦截等）走 info。 */
function showToast(message: string, tone: ToastTone = "success"): void {
  toastQueue.push(message, tone);
}

function dismissToast(id: number): void {
  toastQueue.dismiss(id);
}

/**
 * 刷新左侧任务列表。
 * silent = true 时不置加载态、不清错误横幅，失败也不弹错，用于构建轮询与新建后的静默补刷。
 */
async function loadJobs(path: string, silent = false): Promise<void> {
  const token = ++jobsToken;
  if (!silent) {
    loadingJobs.value = true;
    errorMessage.value = "";
  }
  try {
    const useView = path === "" && activeView.value !== "";
    const result = useView
      ? await jenkinsApi.jobs("", activeView.value)
      : await jenkinsApi.jobs(path, "");
    if (token !== jobsToken) return;

    jobs.value = result.jobs;
    folder.value = path;

    const current = selectedJob.value?.fullName;
    const stillThere = current
      ? result.jobs.some((job) => job.fullName === current)
      : false;
    if (!stillThere) {
      const first = result.jobs.find((job) => !job.isFolder);
      if (first) {
        void selectJob(first, silent);
      } else {
        selectedJob.value = null;
        builds.value = [];
        selectedBuild.value = null;
        stageMatrix.value = null;
      }
    }
  } catch (error) {
    if (token === jobsToken && !silent) setError(error);
  } finally {
    // 静默请求也可能持有最新 token，必须一并关掉上一个请求留下的加载态。
    if (token === jobsToken) loadingJobs.value = false;
  }
}

async function loadViews(): Promise<void> {
  try {
    const result = await jenkinsApi.views();
    views.value = result.views.length
      ? result.views
      : [{ name: "", description: "", url: "" }];
  } catch {
    // 旧 Jenkins 或权限受限时回退为单个「全部任务」。
    views.value = [{ name: "", description: "", url: "" }];
  }

  if (!views.value.some((item) => item.name === activeView.value)) {
    const preferred =
      views.value.find((item) => item.name.toLowerCase() === "all") ?? views.value[0];
    activeView.value = preferred.name;
  }
}

async function activateView(name: string): Promise<void> {
  activeView.value = name;
  folder.value = "";
  selectedJob.value = null;
  selectedBuild.value = null;
  stageMatrix.value = null;
  builds.value = [];
  pendingBuild.value = null;
  view.value = "jobs";
  await loadJobs("");
}

async function loadOverview(): Promise<void> {
  loadingView.value = true;
  errorMessage.value = "";
  try {
    overview.value = await jenkinsApi.overview();
  } catch (error) {
    setError(error);
  } finally {
    loadingView.value = false;
  }
}

async function loadQueue(): Promise<void> {
  loadingView.value = true;
  errorMessage.value = "";
  try {
    const result = await jenkinsApi.queue();
    queueItems.value = result.items;
  } catch (error) {
    setError(error);
  } finally {
    loadingView.value = false;
  }
}

async function loadNodes(): Promise<void> {
  loadingView.value = true;
  errorMessage.value = "";
  try {
    const result = await jenkinsApi.nodes();
    nodes.value = result.nodes;
  } catch (error) {
    setError(error);
  } finally {
    loadingView.value = false;
  }
}

async function switchView(next: "overview" | "jobs" | "queue" | "nodes"): Promise<void> {
  view.value = next;
  if (next === "overview") await loadOverview();
  else if (next === "queue") await loadQueue();
  else if (next === "nodes") await loadNodes();
  else if (next === "jobs") {
    if (!views.value.length) await loadViews();
    if (!jobs.value.length) await loadJobs("");
  }
}

async function cancelQueueItem(id: number): Promise<void> {
  if (!window.confirm(t("cancelConfirm", { id }))) return;
  try {
    await jenkinsApi.cancelQueue(id);
    showToast(t("cancelSuccess"));
    await loadQueue();
  } catch (error) {
    setError(error);
  }
}

async function loadIdentity(): Promise<void> {
  errorMessage.value = "";
  try {
    const result = await jenkinsApi.whoami();
    readOnly.value = result.readOnly;
    identity.value = {
      name: result.user?.name ?? "",
      version: result.user?.version ?? "",
      mode: result.user?.mode ?? "",
    };
    await Promise.all([loadOverview(), loadViews()]);
  } catch (error) {
    setError(error);
  }
}

/**
 * 加载任务详情 + 构建列表。
 * silent = true 时不置加载态、不清错误横幅，用于触发构建后的快速补刷。
 */
async function selectJob(
  job: Pick<JobSummary, "fullName">,
  silent = false,
): Promise<void> {
  const token = ++selectToken;
  view.value = "jobs";
  if (!silent) {
    selectedBuild.value = null;
    stageMatrix.value = null;
    loadingJob.value = true;
    loadingBuilds.value = true;
    errorMessage.value = "";
  }
  try {
    const [detail, list] = await Promise.all([
      jenkinsApi.job(job.fullName),
      jenkinsApi.builds(job.fullName, 20),
    ]);
    if (token !== selectToken) return;
    selectedJob.value = detail;
    builds.value = list.builds;
    settlePendingBuild(job.fullName, list.builds);
  } catch (error) {
    if (token === selectToken && !silent) setError(error);
  } finally {
    // 静默补刷可能覆盖上一个非静默请求的 token，这里必须负责收尾。
    if (token === selectToken) {
      loadingJob.value = false;
      loadingBuilds.value = false;
    }
  }
}

async function refreshJob(silent = false): Promise<void> {
  if (!selectedJob.value) {
    await loadJobs(folder.value);
    return;
  }
  await selectJob({ fullName: selectedJob.value.fullName }, silent);
  autoRefresher?.restart();
}

async function openBuild(number: number, rememberReturn = true): Promise<void> {
  if (!selectedJob.value) return;
  if (rememberReturn) {
    buildReturnView.value = view.value === "stages" ? "stages" : "jobs";
  }
  loadingBuilds.value = true;
  errorMessage.value = "";
  try {
    selectedBuild.value = await jenkinsApi.build(selectedJob.value.fullName, number);
    view.value = "build";
  } catch (error) {
    setError(error);
  } finally {
    loadingBuilds.value = false;
  }
}

async function refreshBuild(): Promise<void> {
  if (!selectedBuild.value || !selectedJob.value) return;
  await openBuild(selectedBuild.value.number, false);
}

function closeBuild(): void {
  view.value = buildReturnView.value === "stages" ? "stages" : "jobs";
  if (view.value === "stages") refreshStages();
}

async function loadStages(job: string): Promise<void> {
  const token = ++stageToken;
  loadingStages.value = true;
  errorMessage.value = "";
  try {
    const matrix = await jenkinsApi.stages(job, 10);
    if (token !== stageToken) return;
    stageMatrix.value = matrix;
  } catch (error) {
    if (token !== stageToken) return;
    stageMatrix.value = null;
    setError(error);
  } finally {
    if (token === stageToken) loadingStages.value = false;
  }
}

function pageHidden(): boolean {
  return typeof document !== "undefined" && document.hidden;
}

/** 判定刚触发的构建是否已出现在列表里；settled / expired / stale 都清掉跟踪。 */
function settlePendingBuild(job: string, list: BuildSummary[]): void {
  const pending = pendingBuild.value;
  if (!pending) return;
  const status = evaluatePendingBuild(pending, job, list);
  if (status === "watching") return;
  pendingBuild.value = null;
  // 新构建真正出现后补刷任务列表：此时圆点与 #构建号 才会更新。
  if (status === "settled") void loadJobs(folder.value, true);
}

/** 静默刷新构建列表 + 任务详情：不置 loading，失败不弹错误。 */
async function pollBuilds(): Promise<void> {
  const job = selectedJob.value?.fullName;
  if (!job || view.value !== "jobs" || pageHidden()) return;
  try {
    const [detail, list] = await Promise.all([
      jenkinsApi.job(job),
      jenkinsApi.builds(job, 20),
    ]);
    if (view.value !== "jobs" || selectedJob.value?.fullName !== job) return;
    selectedJob.value = detail;
    builds.value = list.builds;
    settlePendingBuild(job, list.builds);
  } catch {
    // 自动刷新失败保持现有数据，下一轮继续重试。
  }
}

/** 静默刷新阶段矩阵；不抢占 stageToken，手动刷新会让本轮旧响应作废。 */
async function pollStages(): Promise<void> {
  const job = selectedJob.value?.fullName;
  if (!job || view.value !== "stages" || pageHidden()) return;
  const token = stageToken;
  try {
    const matrix = await jenkinsApi.stages(job, 10);
    if (
      token !== stageToken ||
      view.value !== "stages" ||
      selectedJob.value?.fullName !== job
    ) {
      return;
    }
    stageMatrix.value = matrix;
  } catch {
    // 同上，静默重试。
  }
}

function stopAutoRefresh(): void {
  autoRefresher?.stop();
  autoRefresher = null;
}

function startAutoRefresh(mode: "builds" | "stages"): void {
  const refresher = createAutoRefresher({
    intervalMs:
      mode === "builds" ? AUTO_REFRESH_INTERVAL_MS : STAGE_AUTO_REFRESH_INTERVAL_MS,
    task: mode === "builds" ? pollBuilds : pollStages,
    shouldContinue: () => autoRefreshMode.value === mode,
  });
  autoRefresher = refresher;
  refresher.start();
}

watch(autoRefreshMode, (mode, previous) => {
  stopAutoRefresh();
  if (mode) {
    startAutoRefresh(mode);
    return;
  }
  // 构建结束、轮询停止时补一次静默刷新，让列表圆点与构建号收敛到最终状态。
  if (previous === "builds" && view.value === "jobs") {
    void loadJobs(folder.value, true);
  }
});

async function openStages(): Promise<void> {
  if (!selectedJob.value || !isPipeline.value) return;
  view.value = "stages";
  await loadStages(selectedJob.value.fullName);
}

async function refreshStages(): Promise<void> {
  if (!selectedJob.value) return;
  await loadStages(selectedJob.value.fullName);
  autoRefresher?.restart();
}

async function triggerBuild(params: Record<string, unknown> = {}): Promise<void> {
  const job = selectedJob.value;
  if (!job) return;
  try {
    const result = await jenkinsApi.trigger(job.fullName, params);
    showToast(t("buildQueued", { queueId: result.queueId || "?" }));
    // 参数化构建要排队后才分配构建号：先立即补刷一次，再跟踪到新构建出现为止。
    pendingBuild.value = createPendingBuild(job.fullName, job.lastBuild?.number ?? 0);
    void refreshJob(true);
    // 列表里的圆点与构建号也要跟着变，否则左侧看起来像没触发。
    void loadJobs(folder.value, true);
    window.setTimeout(() => {
      if (pendingBuild.value?.job === job.fullName) {
        void refreshJob(true);
        void loadJobs(folder.value, true);
      }
    }, 800);
  } catch (error) {
    setError(error);
  }
}

async function stopBuild(number: number): Promise<void> {
  if (!selectedJob.value) return;
  if (!window.confirm(t("stopConfirm", { number }))) return;
  try {
    await jenkinsApi.stop(selectedJob.value.fullName, number);
    showToast(t("stopSuccess"));
    window.setTimeout(() => {
      void refreshJob();
      if (view.value === "build") void refreshBuild();
    }, 800);
  } catch (error) {
    setError(error);
  }
}

async function stopLastBuild(): Promise<void> {
  const number = selectedJob.value?.lastBuild?.number;
  if (number) await stopBuild(number);
}

function submitParams(params: Record<string, unknown>): void {
  showParams.value = false;
  void triggerBuild(params);
}

/** 只读连接下所有 CRUD 入口统一拦截。 */
function guardWrite(): boolean {
  if (!readOnly.value) return true;
  showToast(t("readOnlyManageHint"), "info");
  return false;
}

async function loadViewJobPool(): Promise<void> {
  try {
    const result = await jenkinsApi.jobs("", "");
    viewJobPool.value = result.jobs
      .filter((job) => !job.isFolder)
      .map((job) => job.fullName);
  } catch {
    viewJobPool.value = jobs.value
      .filter((job) => !job.isFolder)
      .map((job) => job.fullName);
  }
}

async function selectByFullName(fullName: string): Promise<void> {
  const entry = jobs.value.find((job) => job.fullName === fullName);
  if (entry) await selectJob(entry);
}

async function openCreateJob(): Promise<void> {
  if (!guardWrite()) return;
  errorMessage.value = "";
  jobEditor.value = { mode: "create" };
}

async function openEditJob(): Promise<void> {
  if (!selectedJob.value || !guardWrite()) return;
  editorBusy.value = true;
  errorMessage.value = "";
  try {
    const config = await jenkinsApi.jobConfig(selectedJob.value.fullName);
    jobEditor.value = { mode: "edit", config };
  } catch (error) {
    setError(error);
  } finally {
    editorBusy.value = false;
  }
}

type JoinViewResult = "added" | "skipped" | "failed";

/**
 * 新建任务后把它写进当前命名视图。
 * Jenkins 的 ListView 是显式任务集合，不写回的话刷新也看不到新任务。
 * 只做追加：以服务端刚读到的 jobNames 为基线，不动描述、列与过滤规则。
 */
async function joinActiveView(
  fullName: string,
  createdIn: string,
): Promise<JoinViewResult> {
  if (
    !needsJoinView({
      folder: createdIn,
      activeView: activeView.value,
      fullName,
      jobs: jobs.value,
    })
  ) {
    return "skipped";
  }

  const view = activeView.value;
  try {
    const config = await jenkinsApi.viewConfig(view);
    if (resolveJoinAction(config.fields) !== "add" || !config.fields) {
      return "failed";
    }
    await jenkinsApi.updateView(view, {
      jobNames: appendJobName(config.fields.jobNames, fullName),
    });
    // 请求期间用户可能切走视图，此时不再按旧视图校验。
    if (activeView.value !== view) return "added";
    await loadJobs("", true);
    return containsJob(jobs.value, fullName) ? "added" : "failed";
  } catch {
    // 任务已经创建成功，写视图失败不回滚，交给提示引导用户手动添加。
    return "failed";
  }
}

async function createJob(payload: CreateJobPayload): Promise<void> {
  editorBusy.value = true;
  errorMessage.value = "";
  try {
    const result = await jenkinsApi.createJob(payload);
    const fullName = result.fullName ?? payload.name;
    jobEditor.value = null;

    // 先刷新列表：正则视图或全部视图能立即包含新任务，避免多余的写请求。
    await loadJobs(folder.value, true);
    const join = await joinActiveView(fullName, payload.folder);
    await selectByFullName(fullName);

    if (join === "added") {
      showToast(
        t("jobCreatedInView", { name: fullName, view: activeView.value }),
      );
    } else if (join === "failed") {
      // 任务建好了但没进视图，属于部分成功：用 info 语气提示，别给绿色勾。
      showToast(
        t("jobCreatedNotInView", { name: fullName, view: activeView.value }),
        "info",
      );
    } else {
      showToast(t("jobCreated", { name: fullName }));
    }
  } catch (error) {
    setError(error);
  } finally {
    editorBusy.value = false;
  }
}

async function updateJob(fields: JobUpdateFields): Promise<void> {
  if (!selectedJob.value) return;
  const previous = selectedJob.value.fullName;
  editorBusy.value = true;
  errorMessage.value = "";
  try {
    const result = await jenkinsApi.updateJob(previous, fields);
    jobEditor.value = null;
    showToast(t("jobUpdated"));
    await loadJobs(folder.value);
    await selectByFullName(result.fullName ?? previous);
  } catch (error) {
    setError(error);
  } finally {
    editorBusy.value = false;
  }
}

async function saveJobXml(xml: string): Promise<void> {
  if (!selectedJob.value) return;
  const fullName = selectedJob.value.fullName;
  editorBusy.value = true;
  errorMessage.value = "";
  try {
    await jenkinsApi.saveJob(fullName, xml);
    jobEditor.value = null;
    showToast(t("jobUpdated"));
    await loadJobs(folder.value);
    await selectByFullName(fullName);
  } catch (error) {
    setError(error);
  } finally {
    editorBusy.value = false;
  }
}

async function openCreateView(): Promise<void> {
  if (!guardWrite()) return;
  errorMessage.value = "";
  await loadViewJobPool();
  viewEditor.value = { mode: "create" };
}

async function openEditView(): Promise<void> {
  if (!canManageView.value || !guardWrite()) return;
  const name = activeView.value;
  editorBusy.value = true;
  errorMessage.value = "";
  try {
    const [config] = await Promise.all([jenkinsApi.viewConfig(name), loadViewJobPool()]);
    const view = views.value.find((item) => item.name === name) ?? {
      name,
      description: "",
      url: "",
    };
    viewEditor.value = { mode: "edit", view, config };
  } catch (error) {
    setError(error);
  } finally {
    editorBusy.value = false;
  }
}

async function createView(payload: CreateViewPayload): Promise<void> {
  editorBusy.value = true;
  errorMessage.value = "";
  try {
    await jenkinsApi.createView(payload);
    viewEditor.value = null;
    showToast(t("viewCreated", { name: payload.name }));
    await loadViews();
    activeView.value = payload.name;
    folder.value = "";
    await loadJobs("");
  } catch (error) {
    setError(error);
  } finally {
    editorBusy.value = false;
  }
}

async function updateView(fields: ViewUpdateFields): Promise<void> {
  const name = activeView.value;
  editorBusy.value = true;
  errorMessage.value = "";
  try {
    const result = await jenkinsApi.updateView(name, fields);
    const nextName = result.fullName || name;
    viewEditor.value = null;
    showToast(
      nextName !== name ? t("viewRenamed", { name: nextName }) : t("viewUpdated"),
    );
    await loadViews();
    if (nextName !== name) {
      // 视图改名后 URL 变了，必须先切到新名再拉任务，否则会 404。
      activeView.value = nextName;
      folder.value = "";
    }
    await loadJobs("");
  } catch (error) {
    setError(error);
  } finally {
    editorBusy.value = false;
  }
}

async function saveViewXml(xml: string): Promise<void> {
  const name = activeView.value;
  editorBusy.value = true;
  errorMessage.value = "";
  try {
    await jenkinsApi.saveView(name, xml);
    viewEditor.value = null;
    showToast(t("viewUpdated"));
    await loadViews();
    await loadJobs("");
  } catch (error) {
    setError(error);
  } finally {
    editorBusy.value = false;
  }
}

function openDeleteJob(): void {
  if (!selectedJob.value || !guardWrite()) return;
  deleteTarget.value = {
    kind: "job",
    name: selectedJob.value.name,
    fullName: selectedJob.value.fullName,
  };
}

function openDeleteView(): void {
  if (!canManageView.value || !guardWrite()) return;
  deleteTarget.value = { kind: "view", name: activeView.value };
}

async function confirmDelete(): Promise<void> {
  if (!deleteTarget.value) return;
  const target = deleteTarget.value;
  editorBusy.value = true;
  errorMessage.value = "";
  try {
    if (target.kind === "job") {
      await jenkinsApi.deleteJob(target.fullName ?? target.name);
      showToast(t("jobDeleted"));
      deleteTarget.value = null;
      selectedJob.value = null;
      builds.value = [];
      selectedBuild.value = null;
      stageMatrix.value = null;
      await loadJobs(folder.value);
    } else {
      await jenkinsApi.deleteView(target.name);
      showToast(t("viewDeleted"));
      deleteTarget.value = null;
      await loadViews();
      await loadJobs("");
    }
  } catch (error) {
    setError(error);
  } finally {
    editorBusy.value = false;
  }
}

async function refreshCurrent(): Promise<void> {
  if (view.value === "build") {
    await refreshBuild();
  } else if (view.value === "stages") {
    refreshStages();
  } else if (view.value === "overview") {
    await loadOverview();
  } else if (view.value === "queue") {
    await loadQueue();
  } else if (view.value === "nodes") {
    await loadNodes();
  } else {
    await refreshJob();
  }
}

function applyTheme(): void {
  const plugin = window.dbxPlugin;
  if (!plugin) return;
  const theme = plugin.theme;
  if (theme?.appearance) {
    document.documentElement.dataset.theme = theme.appearance;
  }
  for (const [key, value] of Object.entries(theme?.tokens ?? {})) {
    if (/^--(color|radius|font)-/.test(key)) {
      document.documentElement.style.setProperty(key, value);
    }
  }
}

function handleEnv(): void {
  syncLocale(window.dbxPlugin?.locale);
  applyTheme();
}

onMounted(async () => {
  const plugin = window.dbxPlugin;
  if (!plugin) {
    errorMessage.value = "window.dbxPlugin is undefined";
    return;
  }
  await plugin.ready;
  setConnectionId(plugin.context?.connectionId);
  connectionName.value = String(plugin.context?.connectionName ?? plugin.context?.name ?? "");
  syncLocale(plugin.locale);
  applyTheme();
  window.addEventListener("dbx-plugin-env", handleEnv);
  await loadIdentity();
});

onBeforeUnmount(() => {
  window.removeEventListener("dbx-plugin-env", handleEnv);
  toastQueue.clear();
  pendingBuild.value = null;
  stopAutoRefresh();
});
</script>

<template>
  <div class="app">
    <header class="topbar">
      <div class="brand">
        <img class="logo" :src="logoUrl" alt="" />
        <div class="brand-text">
          <strong>{{ t("appTitle") }}</strong>
          <span class="subtle" :title="connectionName">{{ connectionLabel }}</span>
        </div>
      </div>

      <div class="topbar-right">
        <span v-if="identity" class="who">
          {{ identity.name || t("anonymous") }}
          <template v-if="identity.version"> · {{ identity.version }}</template>
        </span>
        <span class="chip" :data-write="!readOnly">
          {{ readOnly ? t("readOnlyBadge") : t("writeBadge") }}
        </span>
        <button type="button" class="refresh" @click="refreshCurrent">{{ t("refresh") }}</button>
      </div>
    </header>

    <div class="body">
      <nav class="sidebar">
        <button
          type="button"
          class="nav-item"
          :class="{ active: view === 'overview' }"
          @click="switchView('overview')"
        >
          <svg viewBox="0 0 16 16" aria-hidden="true">
            <path d="M3 3h4v4H3zM9 3h4v4H9zM3 9h4v4H3zM9 9h4v4H9z" fill="none" stroke="currentColor" stroke-width="1.3" />
          </svg>
          <span>{{ t("overview") }}</span>
        </button>
        <button
          type="button"
          class="nav-item"
          :class="{ active: view === 'jobs' || view === 'build' || view === 'stages' }"
          @click="switchView('jobs')"
        >
          <svg viewBox="0 0 16 16" aria-hidden="true">
            <path d="M3 4h10M3 8h10M3 12h6" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
          </svg>
          <span>{{ t("jobs") }}</span>
        </button>
        <button
          type="button"
          class="nav-item"
          :class="{ active: view === 'queue' }"
          @click="switchView('queue')"
        >
          <svg viewBox="0 0 16 16" aria-hidden="true">
            <path d="M3 4h10M3 8h10M3 12h10" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
          </svg>
          <span>{{ t("queue") }}</span>
        </button>
        <button
          type="button"
          class="nav-item"
          :class="{ active: view === 'nodes' }"
          @click="switchView('nodes')"
        >
          <svg viewBox="0 0 16 16" aria-hidden="true">
            <path d="M8 2v4M8 10v4M4 8H2M14 8h-2M8 6a2 2 0 1 0 0 4 2 2 0 0 0 0-4z" fill="none" stroke="currentColor" stroke-width="1.3" />
          </svg>
          <span>{{ t("nodes") }}</span>
        </button>
      </nav>

      <div class="workspace">
        <div v-if="errorMessage" class="banner error">
          <span>{{ errorMessage }}</span>
          <button type="button" @click="loadIdentity">{{ t("retry") }}</button>
        </div>

        <OverviewView
          v-if="view === 'overview'"
          :overview="overview"
          :loading="loadingView"
          @refresh="loadOverview"
        />

        <QueueView
          v-else-if="view === 'queue'"
          :items="queueItems"
          :loading="loadingView"
          :read-only="readOnly"
          :locale="locale"
          @cancel="cancelQueueItem"
          @refresh="loadQueue"
        />

        <NodesView
          v-else-if="view === 'nodes'"
          :nodes="nodes"
          :loading="loadingView"
          @refresh="loadNodes"
        />

        <StageView
          v-else-if="view === 'stages' && selectedJob"
          :job="selectedJob.fullName"
          :matrix="stageMatrix"
          :loading="loadingStages"
          :locale="locale"
          :auto-refreshing="autoRefreshing"
          @back="view = 'jobs'"
          @refresh="refreshStages"
          @open-build="openBuild"
        />

        <div v-else-if="view === 'build' && selectedBuild && selectedJob" class="build-wrap">
          <BuildView
            :build="selectedBuild"
            :job-full-name="selectedJob.fullName"
            :read-only="readOnly"
            :locale="locale"
            @back="closeBuild"
            @stop="stopBuild"
          />
        </div>

        <div v-else class="jobs-wrap">
          <JobList
            :jobs="jobs"
            :selected="selectedJob?.fullName ?? ''"
            :folder="folder"
            :loading="loadingJobs"
            :views="views"
            :active-view="activeView"
            :read-only="readOnly"
            @select="selectJob"
            @open-folder="loadJobs($event.fullName)"
            @navigate="loadJobs"
            @select-view="activateView"
            @refresh="loadJobs(folder)"
            @create-job="openCreateJob"
            @create-view="openCreateView"
            @edit-view="openEditView"
            @delete-view="openDeleteView"
          />

          <JobView
            v-if="selectedJob"
            :job="selectedJob"
            :builds="builds"
            :loading-builds="loadingJob || loadingBuilds"
            :read-only="readOnly"
            :is-pipeline="isPipeline"
            :locale="locale"
            :auto-refreshing="autoRefreshing"
            @build="triggerBuild()"
            @build-with-params="showParams = true"
            @stop="stopLastBuild"
            @open-build="openBuild"
            @open-stages="openStages"
            @refresh="refreshJob"
            @edit-job="openEditJob"
            @delete-job="openDeleteJob"
          />
          <div v-else class="placeholder">
            <p>{{ t("selectJob") }}</p>
          </div>
        </div>
      </div>
    </div>

    <ParamDialog
      v-if="showParams && selectedJob"
      :job="selectedJob"
      @submit="submitParams"
      @cancel="showParams = false"
    />

    <JobEditor
      v-if="jobEditor"
      :mode="jobEditor.mode"
      :folder="folder"
      :config="jobEditor.mode === 'edit' ? jobEditor.config : null"
      :job="selectedJob"
      :busy="editorBusy"
      @create="createJob"
      @update="updateJob"
      @save-xml="saveJobXml"
      @cancel="jobEditor = null"
    />

    <ViewEditor
      v-if="viewEditor"
      :mode="viewEditor.mode"
      :view="viewEditor.mode === 'edit' ? viewEditor.view : null"
      :config="viewEditor.mode === 'edit' ? viewEditor.config : null"
      :available-jobs="viewJobPool"
      :existing-names="viewNames"
      :busy="editorBusy"
      @create="createView"
      @update="updateView"
      @save-xml="saveViewXml"
      @cancel="viewEditor = null"
    />

    <ConfirmDelete
      v-if="deleteTarget"
      :target="deleteTarget.name"
      :kind="deleteTarget.kind"
      :full-name="deleteTarget.fullName"
      :busy="editorBusy"
      @confirm="confirmDelete"
      @cancel="deleteTarget = null"
    />

    <ToastHost :items="toasts" @dismiss="dismissToast" />
  </div>
</template>

<style scoped>
.app {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  height: 56px;
  padding: 0 18px;
  border-bottom: 1px solid var(--border);
  background: var(--panel);
  flex: none;
}

.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.logo {
  width: 28px;
  height: 28px;
  border-radius: 8px;
}

.brand-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
  line-height: 1.25;
}

.brand-text strong {
  font-size: 14px;
}

.subtle {
  color: var(--muted);
  font-size: 12px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: none;
}

.who {
  color: var(--muted);
  font-size: 12px;
}

.chip {
  padding: 4px 10px;
  border-radius: 999px;
  background: var(--accent-soft);
  color: var(--accent);
  font-size: 12px;
  font-weight: 600;
}

.chip[data-write="true"] {
  background: color-mix(in srgb, var(--success) 14%, transparent);
  color: var(--success);
}

.refresh {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 6px 11px;
  background: transparent;
  color: var(--text);
  font-size: 12px;
  font-weight: 600;
}

.refresh:hover {
  border-color: var(--accent);
  color: var(--accent);
}

.body {
  display: flex;
  flex: 1;
  min-height: 0;
}

.sidebar {
  width: 196px;
  flex: none;
  padding: 12px;
  border-right: 1px solid var(--border);
  background: var(--panel);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 9px;
  width: 100%;
  padding: 9px 10px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--muted);
  text-align: left;
}

.nav-item svg {
  width: 16px;
  height: 16px;
  flex: none;
}

.nav-item.active {
  background: var(--accent-soft);
  color: var(--accent);
  font-weight: 600;
}

.nav-item:disabled {
  opacity: 0.72;
  cursor: default;
}

.soon {
  margin-left: auto;
  font-size: 9px;
  padding: 2px 5px;
  border-radius: 4px;
  background: var(--panel-soft);
  color: var(--muted);
}

.workspace {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 16px;
  font-size: 12px;
  flex: none;
}

.banner.error {
  background: color-mix(in srgb, var(--danger) 12%, transparent);
  color: var(--danger);
}

.banner button {
  border: 1px solid currentColor;
  border-radius: var(--radius-sm);
  background: transparent;
  color: inherit;
  padding: 3px 9px;
  font-size: 12px;
}

.jobs-wrap {
  flex: 1;
  min-height: 0;
  display: flex;
}

.build-wrap {
  flex: 1;
  min-height: 0;
  display: flex;
}

.placeholder {
  flex: 1;
  display: grid;
  place-items: center;
  color: var(--muted);
  font-size: 13px;
}
</style>
