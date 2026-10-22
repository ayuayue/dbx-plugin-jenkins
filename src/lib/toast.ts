/** 操作提示的类型：写操作成功用 success，纯提示（如只读拦截）用 info。 */
export type ToastTone = "success" | "info";

export interface ToastItem {
  id: number;
  text: string;
  tone: ToastTone;
}

/** 提示停留时长：与旧的内联横幅一致，4s 后自动消失。 */
export const TOAST_DURATION_MS = 4000;

/** 同时最多显示几条：超出后挤掉最早的一条，避免连点操作把界面堆满。 */
export const TOAST_MAX_VISIBLE = 3;

export interface ToastQueueOptions {
  durationMs?: number;
  maxVisible?: number;
  /** 列表每次变化回调一次，UI 层据此同步到响应式状态。 */
  onChange?: (items: ToastItem[]) => void;
}

export interface ToastQueue {
  /** 入队一条提示，返回其 id；空白文本直接忽略并返回 null。 */
  push(text: string, tone?: ToastTone): number | null;
  /** 立即移除指定提示（点击关闭）。 */
  dismiss(id: number): void;
  /** 清空全部提示并取消所有计时器（组件卸载时调用）。 */
  clear(): void;
  /** 当前提示快照，按入队顺序。 */
  items(): ToastItem[];
  size(): number;
}

/**
 * 操作提示队列。
 * 定时器用全局 setTimeout，便于在 Node 测试环境注入 fake timers。
 */
export function createToastQueue(options: ToastQueueOptions = {}): ToastQueue {
  const durationMs = Math.max(500, options.durationMs ?? TOAST_DURATION_MS);
  const maxVisible = Math.max(1, options.maxVisible ?? TOAST_MAX_VISIBLE);

  let items: ToastItem[] = [];
  let nextId = 1;
  const timers = new Map<number, ReturnType<typeof setTimeout>>();

  function emit(): void {
    options.onChange?.(items.map((item) => ({ ...item })));
  }

  function cancelTimer(id: number): void {
    const timer = timers.get(id);
    if (timer === undefined) return;
    clearTimeout(timer);
    timers.delete(id);
  }

  function schedule(id: number): void {
    cancelTimer(id);
    timers.set(
      id,
      setTimeout(() => remove(id), durationMs),
    );
  }

  function remove(id: number): void {
    cancelTimer(id);
    const next = items.filter((item) => item.id !== id);
    if (next.length === items.length) return;
    items = next;
    emit();
  }

  return {
    push(text, tone = "success") {
      const trimmed = (text ?? "").trim();
      if (!trimmed) return null;

      // 同一句提示连点时不叠加，只把倒计时重置。
      const existing = items.find((item) => item.text === trimmed);
      if (existing) {
        schedule(existing.id);
        return existing.id;
      }

      const item: ToastItem = { id: nextId, text: trimmed, tone };
      nextId += 1;
      items = [...items, item];
      schedule(item.id);

      while (items.length > maxVisible) {
        const dropped = items[0];
        cancelTimer(dropped.id);
        items = items.slice(1);
      }

      emit();
      return item.id;
    },

    dismiss(id) {
      remove(id);
    },

    clear() {
      items.forEach((item) => cancelTimer(item.id));
      if (items.length === 0) return;
      items = [];
      emit();
    },

    items() {
      return items.map((item) => ({ ...item }));
    },

    size() {
      return items.length;
    },
  };
}
