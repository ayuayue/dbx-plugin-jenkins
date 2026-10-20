/** 把滚轮事件换算成横向滚动量时，行模式（DOM_DELTA_LINE）每行按 16px 估算。 */
const LINE_HEIGHT = 16;

/** 判定「可横向滚动」的最小余量，规避亚像素误差。 */
const OVERFLOW_EPSILON = 1;

export interface TabWheelInput {
  deltaX: number;
  deltaY: number;
  /** WheelEvent.deltaMode：0=像素，1=行，2=页。 */
  deltaMode: number;
  scrollLeft: number;
  scrollWidth: number;
  clientWidth: number;
}

/**
 * 把鼠标滚轮换算成 Tab 条需要横向滚动的像素数。
 * 返回 0 表示不拦截，交回浏览器默认行为（无法滚动、已到边界、或本就是横向手势）。
 */
export function tabWheelDelta(input: TabWheelInput): number {
  const { deltaX, deltaY, deltaMode, scrollLeft, scrollWidth, clientWidth } = input;

  const maxScroll = scrollWidth - clientWidth;
  if (maxScroll <= OVERFLOW_EPSILON) return 0;

  // 触控板横向滑动/Shift+滚轮：让元素按原生横向滚动处理，避免方向打架。
  if (Math.abs(deltaX) > Math.abs(deltaY)) return 0;

  if (!Number.isFinite(deltaY) || deltaY === 0) return 0;

  const raw =
    deltaMode === 1 ? deltaY * LINE_HEIGHT : deltaMode === 2 ? deltaY * clientWidth : deltaY;
  if (!Number.isFinite(raw) || raw === 0) return 0;

  const next = Math.min(Math.max(scrollLeft + raw, 0), maxScroll);
  return next - scrollLeft;
}
