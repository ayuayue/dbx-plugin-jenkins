import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { TOAST_DURATION_MS, TOAST_MAX_VISIBLE, createToastQueue } from "./toast";

describe("createToastQueue", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("pushes a toast with a stable id and the default success tone", () => {
    const queue = createToastQueue();

    const id = queue.push("任务已删除");

    expect(id).toBeTypeOf("number");
    expect(queue.items()).toEqual([{ id, text: "任务已删除", tone: "success" }]);
    expect(queue.size()).toBe(1);
  });

  it("keeps the info tone and trims the text", () => {
    const queue = createToastQueue();

    queue.push("  请关闭只读模式  ", "info");

    expect(queue.items()[0]).toMatchObject({ text: "请关闭只读模式", tone: "info" });
  });

  it("ignores empty or blank-only text", () => {
    const queue = createToastQueue();

    expect(queue.push("")).toBeNull();
    expect(queue.push("   ")).toBeNull();
    expect(queue.size()).toBe(0);
  });

  it("auto-dismisses after the default duration", () => {
    const queue = createToastQueue();

    queue.push("已保存");
    vi.advanceTimersByTime(TOAST_DURATION_MS - 1);
    expect(queue.size()).toBe(1);

    vi.advanceTimersByTime(1);
    expect(queue.size()).toBe(0);
  });

  it("dedupes identical text and restarts the countdown instead of stacking", () => {
    const queue = createToastQueue();

    const first = queue.push("任务已删除");
    vi.advanceTimersByTime(TOAST_DURATION_MS - 500);
    const second = queue.push("任务已删除");

    expect(second).toBe(first);
    expect(queue.size()).toBe(1);

    // 计时被重置：距第二次 push 还有 1s 时仍然在。
    vi.advanceTimersByTime(1000);
    expect(queue.size()).toBe(1);

    vi.advanceTimersByTime(TOAST_DURATION_MS);
    expect(queue.size()).toBe(0);
  });

  it("keeps no more than the visible limit and drops the oldest", () => {
    const queue = createToastQueue();

    for (let i = 1; i <= TOAST_MAX_VISIBLE + 1; i += 1) {
      queue.push(`提示 ${i}`);
    }

    expect(queue.size()).toBe(TOAST_MAX_VISIBLE);
    expect(queue.items().map((item) => item.text)).toEqual(["提示 2", "提示 3", "提示 4"]);
  });

  it("dismisses a single toast immediately", () => {
    const queue = createToastQueue();

    const first = queue.push("第一条")!;
    queue.push("第二条");
    queue.dismiss(first);

    expect(queue.items().map((item) => item.text)).toEqual(["第二条"]);
  });

  it("does not touch the remaining toasts when dismissing an unknown id", () => {
    const queue = createToastQueue();

    queue.push("保留我");
    queue.dismiss(999);

    expect(queue.size()).toBe(1);
  });

  it("drops a dismissed toast's timer so it cannot fire twice", () => {
    const queue = createToastQueue();
    const onChange = vi.fn();
    const tracked = createToastQueue({ onChange });

    const id = tracked.push("已排队")!;
    tracked.dismiss(id);
    onChange.mockClear();

    vi.advanceTimersByTime(TOAST_DURATION_MS * 2);
    expect(onChange).not.toHaveBeenCalled();
    expect(queue.size()).toBe(0);
  });

  it("notifies subscribers on push, drop and expiry", () => {
    const onChange = vi.fn();
    const queue = createToastQueue({ onChange });

    queue.push("A");
    expect(onChange).toHaveBeenCalledTimes(1);
    expect(onChange).toHaveBeenLastCalledWith([expect.objectContaining({ text: "A" })]);

    queue.push("B");
    expect(onChange).toHaveBeenCalledTimes(2);

    // 两条同时到期，各自回调一次，最后一次一定是空列表。
    vi.advanceTimersByTime(TOAST_DURATION_MS);
    expect(onChange).toHaveBeenCalledTimes(4);
    expect(onChange).toHaveBeenLastCalledWith([]);
  });

  it("honours a custom duration", () => {
    const queue = createToastQueue({ durationMs: 1000 });

    queue.push("短提示");
    vi.advanceTimersByTime(999);
    expect(queue.size()).toBe(1);

    vi.advanceTimersByTime(1);
    expect(queue.size()).toBe(0);
  });

  it("can re-push the same text after it expired", () => {
    const queue = createToastQueue();

    queue.push("任务已删除");
    vi.advanceTimersByTime(TOAST_DURATION_MS);

    const id = queue.push("任务已删除");
    expect(queue.size()).toBe(1);
    expect(queue.items()[0].id).toBe(id);
  });

  it("clear() removes everything and cancels pending timers", () => {
    const onChange = vi.fn();
    const queue = createToastQueue({ onChange });

    queue.push("A");
    queue.push("B");
    queue.clear();
    expect(queue.size()).toBe(0);

    onChange.mockClear();
    vi.advanceTimersByTime(TOAST_DURATION_MS * 2);
    expect(onChange).not.toHaveBeenCalled();
  });
});
