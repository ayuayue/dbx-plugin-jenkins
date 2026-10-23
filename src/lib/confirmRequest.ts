/**
 * 停止构建 / 取消排队项一律走应用内弹窗确认。
 *
 * 起因：宿主 webview 里的原生 `window.confirm()` 会被静默忽略（直接返回 false），
 * 表现就是点了「停止」毫无反应。这里只负责构造待确认目标并做入参校验，
 * 弹窗的显隐与 busy 状态留在 App.vue。
 */

export interface StopRequest {
  kind: "stop";
  job: string;
  number: number;
}

export interface CancelRequest {
  kind: "cancel";
  id: string;
  jobName: string;
}

export type ConfirmRequest = StopRequest | CancelRequest;

/** Jenkins 的构建号 / 队列 id 都是从 1 开始的正整数。 */
function isPositiveInteger(value: number | null | undefined): value is number {
  return typeof value === "number" && Number.isInteger(value) && value > 0;
}

/** 任务全名或构建号不可用时返回 null，调用方据此不弹窗、不发请求。 */
export function createStopRequest(
  job: string | null | undefined,
  number: number | null | undefined,
): StopRequest | null {
  const fullName = (job ?? "").trim();
  if (!fullName || !isPositiveInteger(number)) return null;
  return { kind: "stop", job: fullName, number };
}

export function createCancelRequest(
  id: number | null | undefined,
  jobName?: string | null,
): CancelRequest | null {
  if (!isPositiveInteger(id)) return null;
  return { kind: "cancel", id: String(id), jobName: (jobName ?? "").trim() };
}

/**
 * 确认后延迟刷新时用：只有当前仍停在同一任务上才刷新，
 * 否则用户中途切了任务会刷错对象。
 */
export function isStopTarget(
  request: ConfirmRequest | null,
  job: string | null | undefined,
): boolean {
  return request?.kind === "stop" && request.job === (job ?? "");
}
