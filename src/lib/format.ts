export type UiLocale = "zh" | "en";

/** 把毫秒时长格式化为 1h 2m / 3m 4s / 12s。 */
export function formatDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) return "—";
  const totalSeconds = Math.floor(ms / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) return `${minutes}m ${seconds}s`;
  return `${seconds}s`;
}

/** 把 Jenkins 的 epoch 毫秒格式化成本地时间。 */
export function formatTimestamp(ms: number, locale: UiLocale = "zh"): string {
  if (!ms || !Number.isFinite(ms)) return "—";
  return new Date(ms).toLocaleString(locale === "zh" ? "zh-CN" : "en-US", {
    hour12: false,
  });
}
