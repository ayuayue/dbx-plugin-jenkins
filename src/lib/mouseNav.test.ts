import { describe, expect, it } from "vitest";
import {
  NAV_DEDUPE_MS,
  NAV_MOUSE_FALLBACK_MS,
  createNavDeduper,
  resolveMouseButton,
} from "./mouseNav";

describe("resolveMouseButton", () => {
  it("侧键 3 是后退、4 是前进", () => {
    expect(resolveMouseButton(3)).toBe("back");
    expect(resolveMouseButton(4)).toBe("forward");
  });

  it("左中右键不参与导航", () => {
    expect(resolveMouseButton(0)).toBeNull();
    expect(resolveMouseButton(1)).toBeNull();
    expect(resolveMouseButton(2)).toBeNull();
  });

  it("越界的按钮值忽略", () => {
    expect(resolveMouseButton(-1)).toBeNull();
    expect(resolveMouseButton(5)).toBeNull();
    expect(resolveMouseButton(Number.NaN)).toBeNull();
  });
});

describe("createNavDeduper", () => {
  it("同方向在窗口内只放行一次", () => {
    const dedupe = createNavDeduper(400);
    expect(dedupe.accept("back", 1000)).toBe(true);
    expect(dedupe.accept("back", 1100)).toBe(false);
    expect(dedupe.accept("back", 1399)).toBe(false);
    expect(dedupe.accept("back", 1401)).toBe(true);
  });

  it("先退再进不受影响", () => {
    const dedupe = createNavDeduper(400);
    expect(dedupe.accept("back", 1000)).toBe(true);
    expect(dedupe.accept("forward", 1010)).toBe(true);
    expect(dedupe.accept("back", 1020)).toBe(true);
  });

  it("窗口以最后一次放行为基准重新计时", () => {
    const dedupe = createNavDeduper(200);
    expect(dedupe.accept("back", 0)).toBe(true);
    expect(dedupe.accept("back", 150)).toBe(false);
    expect(dedupe.accept("back", 201)).toBe(true);
    expect(dedupe.accept("back", 350)).toBe(false);
  });

  it("reset 后立刻可再次放行", () => {
    const dedupe = createNavDeduper(400);
    expect(dedupe.accept("back", 1000)).toBe(true);
    expect(dedupe.accept("back", 1001)).toBe(false);
    dedupe.reset();
    expect(dedupe.accept("back", 1002)).toBe(true);
  });
});

describe("窗口常量", () => {
  it("DOM 兜底延迟要短于去抖窗口", () => {
    expect(NAV_MOUSE_FALLBACK_MS).toBeGreaterThan(0);
    expect(NAV_MOUSE_FALLBACK_MS).toBeLessThan(NAV_DEDUPE_MS);
  });
});
