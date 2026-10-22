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

export interface AnchorRect {
  top: number;
  left: number;
  width: number;
  height: number;
}

export interface PopoverSize {
  width: number;
  height: number;
}

export interface ViewportSize {
  width: number;
  height: number;
}

const POPOVER_GAP = 8;

/** 提交哈希短号：取前 8 位，空值显示破折号。 */
export function shortCommitId(id: string, length = 8): string {
  const trimmed = (id ?? "").trim();
  if (!trimmed) return "—";
  return trimmed.slice(0, Math.max(1, length));
}

/** 提交信息摘要：取首个非空行、压缩空白，超长按码点截断并补省略号。 */
export function commitMessageSummary(message: string, max = 120): string {
  const firstLine =
    (message ?? "")
      .split(/\r?\n/)
      .map((line) => line.replace(/\s+/g, " ").trim())
      .find((line) => line.length > 0) ?? "";

  const chars = Array.from(firstLine);
  if (chars.length <= max) return firstLine;
  return chars.slice(0, max).join("") + "…";
}

/** 浮层定位：优先贴在触发区右侧，越界时左翻/上移，最后钳制在视口内。 */
export function popoverPosition(
  anchor: AnchorRect,
  size: PopoverSize,
  viewport: ViewportSize,
  gap = POPOVER_GAP,
): { top: number; left: number } {
  const maxLeft = viewport.width - size.width - gap;
  const maxTop = viewport.height - size.height - gap;

  let left = anchor.left + anchor.width + gap;
  if (left > maxLeft) {
    const flipped = anchor.left - size.width - gap;
    left = flipped >= gap ? flipped : Math.max(gap, maxLeft);
  }
  left = Math.min(Math.max(left, gap), Math.max(gap, maxLeft));

  let top = anchor.top;
  if (top > maxTop) top = maxTop;
  top = Math.min(Math.max(top, gap), Math.max(gap, maxTop));

  return { top, left };
}
