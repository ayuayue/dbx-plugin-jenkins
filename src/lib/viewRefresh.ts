import type { JobSummary, ListViewFields } from "../types";

/**
 * 是否为「命名视图」：非空且不是主视图 all（Jenkins 里主视图名可能是小写 all）。
 * 只有命名视图才值得把新任务写回，all 视图天然包含所有任务。
 */
export function isNamedView(name?: string | null): boolean {
  const trimmed = (name ?? "").trim();
  return trimmed !== "" && trimmed.toLowerCase() !== "all";
}

/** 任务列表里是否已有该 fullName（精确匹配，避免 demo 命中 demo2）。 */
export function containsJob(
  jobs?: Array<Pick<JobSummary, "fullName">> | null,
  fullName?: string | null,
): boolean {
  const target = (fullName ?? "").trim();
  if (!target) return false;
  return (jobs ?? []).some((job) => job.fullName === target);
}

/**
 * 把任务名追加进视图的 jobNames：保持原顺序、去重、去掉空白项。
 * jobNames 为 null 表示视图没有显式任务，按空集合处理。
 */
export function appendJobName(
  jobNames?: string[] | null,
  fullName?: string | null,
): string[] {
  const base = [...new Set((jobNames ?? []).map((name) => name.trim()).filter(Boolean))];
  const target = (fullName ?? "").trim();
  if (!target || base.includes(target)) return base;
  return [...base, target];
}

export interface JoinViewInput {
  /** 当前浏览的文件夹路径，空串代表根目录。 */
  folder: string;
  /** 当前激活的视图名。 */
  activeView: string;
  /** 新建任务返回的 fullName。 */
  fullName: string;
  /** 刚刷新到的任务列表。 */
  jobs?: Array<Pick<JobSummary, "fullName">> | null;
}

/**
 * 新建任务后是否需要写回当前视图：
 * 只在根目录 + 命名视图 + 列表还看不到新任务时成立。
 * 正则视图若已匹配，刷新就能看到，不再产生写请求。
 */
export function needsJoinView(input: JoinViewInput): boolean {
  if ((input.folder ?? "").trim() !== "") return false;
  if (!isNamedView(input.activeView)) return false;
  if (!(input.fullName ?? "").trim()) return false;
  return !containsJob(input.jobs, input.fullName);
}

export type JoinViewAction = "skip" | "add" | "unsupported";

/**
 * 写回视图的动作判定：
 * - skip：不需要写（已在视图里 / 全部视图 / 文件夹内创建）；
 * - add：ListView，可以追加；
 * - unsupported：非 ListView（fields 为 null），只能提示用户。
 */
export function resolveJoinAction(
  fields?: ListViewFields | null,
  needed: boolean = true,
): JoinViewAction {
  if (!needed) return "skip";
  return fields ? "add" : "unsupported";
}
