export type StageTone = "success" | "danger" | "info" | "muted";

/** 阶段耗时：101ms / 1.5s / 3m 53s；空值用破折号。 */
export function formatStageDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) return "—";
  if (ms < 1000) return `${Math.round(ms)}ms`;
  if (ms < 60000) {
    return `${Number((ms / 1000).toFixed(1))}s`;
  }
  const totalSeconds = Math.floor(ms / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return seconds === 0 ? `${minutes}m` : `${minutes}m ${seconds}s`;
}

/** Jenkins wfapi 阶段状态 → 色调。 */
export function stageTone(status?: string): StageTone {
  switch ((status ?? "").toUpperCase()) {
    case "SUCCESS":
      return "success";
    case "FAILED":
    case "FAILURE":
      return "danger";
    case "IN_PROGRESS":
    case "BUILDING":
      return "info";
    default:
      return "muted";
  }
}

/** 按阶段耗时占最大值的比例，映射到 0.10–0.45 的背景透明度。 */
export function stageIntensity(ms: number, max: number): number {
  if (!Number.isFinite(ms) || !Number.isFinite(max) || ms <= 0 || max <= 0) {
    return 0.1;
  }
  return 0.1 + 0.35 * Math.min(1, ms / max);
}
