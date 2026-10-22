import type {
  BuildDetail,
  BuildSummary,
  ConsoleChunk,
  CreateJobPayload,
  CreateViewPayload,
  JobConfigResponse,
  JobDetail,
  JobSummary,
  JobUpdateFields,
  MutationResult,
  NodeInfo,
  Overview,
  QueueItem,
  StageMatrix,
  TriggerResult,
  ViewConfigResponse,
  ViewInfo,
  ViewUpdateFields,
  WhoAmIResponse,
} from "./types";

let currentConnectionId = "";

export function setConnectionId(id: string | undefined): void {
  currentConnectionId = id ?? "";
}

export function getConnectionId(): string {
  return currentConnectionId;
}

/**
 * 调用本插件 sidecar 的 RPC。
 * 宿主不会自动补 connectionId，必须由 UI 显式携带（与 DBX 官方插件一致）。
 */
export async function invoke<T>(
  method: string,
  params: Record<string, unknown> = {},
  timeoutMs = 30_000,
): Promise<T> {
  const plugin = window.dbxPlugin;
  if (!plugin) {
    throw new Error("DBX host bridge is not available.");
  }
  return plugin.invoke<T>(
    method,
    { connectionId: currentConnectionId, ...params },
    { timeoutMs },
  );
}

export function describeError(error: unknown): string {
  if (error instanceof Error) return error.message;
  if (typeof error === "string") return error;
  try {
    return JSON.stringify(error);
  } catch {
    return "Unknown error";
  }
}

export const jenkinsApi = {
  whoami(): Promise<WhoAmIResponse> {
    return invoke<WhoAmIResponse>("jenkins/whoami", {}, 15_000);
  },

  jobs(folder = "", view = ""): Promise<{ jobs: JobSummary[] }> {
    return invoke<{ jobs: JobSummary[] }>("jenkins/jobs", { folder, view });
  },

  views(): Promise<{ views: ViewInfo[] }> {
    return invoke<{ views: ViewInfo[] }>("jenkins/views", {}, 20_000);
  },

  /**
   * 任务详情。`timeoutMs` 供轮询传更短的超时：
   * 轮询是串行链，单轮挂太久会让后续轮次全部停摆。
   */
  job(job: string, timeoutMs = 30_000): Promise<JobDetail> {
    return invoke<JobDetail>("jenkins/job", { job }, timeoutMs);
  },

  builds(job: string, limit = 20, timeoutMs = 30_000): Promise<{ builds: BuildSummary[] }> {
    return invoke<{ builds: BuildSummary[] }>("jenkins/builds", { job, limit }, timeoutMs);
  },

  build(job: string, number: number, timeoutMs = 30_000): Promise<BuildDetail> {
    return invoke<BuildDetail>("jenkins/build", { job, number }, timeoutMs);
  },

  console(job: string, number: number, start: number): Promise<ConsoleChunk> {
    return invoke<ConsoleChunk>(
      "jenkins/console",
      { job, number, start },
      60_000,
    );
  },

  trigger(job: string, params: Record<string, unknown> = {}): Promise<TriggerResult> {
    return invoke<TriggerResult>("jenkins/trigger", { job, params }, 60_000);
  },

  stop(job: string, number: number): Promise<{ success: boolean }> {
    return invoke<{ success: boolean }>("jenkins/stop", { job, number }, 30_000);
  },

  overview(): Promise<Overview> {
    return invoke<Overview>("jenkins/overview", {}, 30_000);
  },

  queue(timeoutMs = 30_000): Promise<{ items: QueueItem[] }> {
    return invoke<{ items: QueueItem[] }>("jenkins/queue", {}, timeoutMs);
  },

  cancelQueue(id: number): Promise<{ success: boolean }> {
    return invoke<{ success: boolean }>("jenkins/cancelQueue", { id }, 30_000);
  },

  nodes(): Promise<{ nodes: NodeInfo[] }> {
    return invoke<{ nodes: NodeInfo[] }>("jenkins/nodes", {}, 30_000);
  },

  stages(job: string, limit = 10, timeoutMs = 120_000): Promise<StageMatrix> {
    return invoke<StageMatrix>("jenkins/stages", { job, limit }, timeoutMs);
  },

  jobConfig(job: string): Promise<JobConfigResponse> {
    return invoke<JobConfigResponse>("jenkins/jobConfig", { job });
  },

  createJob(payload: CreateJobPayload): Promise<MutationResult> {
    return invoke<MutationResult>("jenkins/createJob", { ...payload }, 60_000);
  },

  updateJob(job: string, fields: JobUpdateFields): Promise<MutationResult> {
    return invoke<MutationResult>("jenkins/updateJob", { job, fields }, 60_000);
  },

  saveJob(job: string, xml: string): Promise<MutationResult> {
    return invoke<MutationResult>("jenkins/saveJob", { job, xml }, 60_000);
  },

  renameJob(job: string, newName: string): Promise<MutationResult> {
    return invoke<MutationResult>("jenkins/renameJob", { job, newName }, 60_000);
  },

  deleteJob(job: string): Promise<MutationResult> {
    return invoke<MutationResult>("jenkins/deleteJob", { job }, 60_000);
  },

  viewConfig(view: string): Promise<ViewConfigResponse> {
    return invoke<ViewConfigResponse>("jenkins/viewConfig", { view });
  },

  createView(payload: CreateViewPayload): Promise<MutationResult> {
    return invoke<MutationResult>("jenkins/createView", { ...payload }, 60_000);
  },

  updateView(view: string, fields: ViewUpdateFields): Promise<MutationResult> {
    return invoke<MutationResult>("jenkins/updateView", { view, fields }, 60_000);
  },

  saveView(view: string, xml: string): Promise<MutationResult> {
    return invoke<MutationResult>("jenkins/saveView", { view, xml }, 60_000);
  },

  deleteView(view: string): Promise<MutationResult> {
    return invoke<MutationResult>("jenkins/deleteView", { view }, 60_000);
  },
};
