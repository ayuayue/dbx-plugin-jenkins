/** 一次导航的方向。 */
export type NavDirection = "back" | "forward";

/**
 * DOM 通道的兜底延迟（毫秒）。
 *
 * 宿主把鼠标侧键当浏览器命令处理时，会先发 `popstate`；我们要空等这么久，
 * 确认没人接这次按键，再自己发起导航。太短会与宿主抢，一次按键退两步；
 * 太长用户会觉得按钮迟钝。
 */
export const NAV_MOUSE_FALLBACK_MS = 90;

/**
 * 同方向导航的去抖窗口（毫秒）。
 *
 * 一次物理按键会先后派发 mousedown 与 mouseup，两个都监听时只执行一次；
 * 快速连按（间隔 150ms 以上）仍然逐次生效——真退两步的人比误触多。
 */
export const NAV_DEDUPE_MS = 150;

/**
 * 鼠标按键号 → 导航方向。
 *
 * 3 / 4 是 Windows 的 XButton1 / XButton2，Chromium 在能派发 DOM 事件时
 * 用它表示后退 / 前进；左中右键（0 / 1 / 2）不参与。
 */
export function resolveMouseButton(button: number): NavDirection | null {
  if (button === 3) return "back";
  if (button === 4) return "forward";
  return null;
}

export interface NavDeduper {
  /** 同方向在窗口内只放行一次；不同方向互不影响。 */
  accept(direction: NavDirection, now?: number): boolean;
  reset(): void;
}

export function createNavDeduper(windowMs: number = NAV_DEDUPE_MS): NavDeduper {
  let lastDirection: NavDirection | null = null;
  let lastAt = 0;

  return {
    accept(direction, now = Date.now()) {
      const inWindow = now - lastAt < windowMs;
      // 时钟回拨（now < lastAt）按不可信处理，也当窗口内，宁可少退一次。
      const sameDirection = lastDirection === direction;
      if (sameDirection && (inWindow || now < lastAt)) return false;
      lastDirection = direction;
      lastAt = now;
      return true;
    },
    reset() {
      lastDirection = null;
      lastAt = 0;
    },
  };
}
