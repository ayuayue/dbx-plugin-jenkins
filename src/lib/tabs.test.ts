import { describe, expect, it } from "vitest";
import { tabWheelDelta } from "./tabs";

/** 横向可滚 300px 的 Tab 条默认场景。 */
const base = {
  deltaX: 0,
  deltaY: 100,
  deltaMode: 0,
  scrollLeft: 0,
  scrollWidth: 600,
  clientWidth: 300,
};

describe("tabWheelDelta", () => {
  it("returns 0 when tabs do not overflow", () => {
    expect(tabWheelDelta({ ...base, scrollWidth: 300 })).toBe(0);
  });

  it("maps vertical wheel to horizontal scroll", () => {
    expect(tabWheelDelta({ ...base, scrollLeft: 100 })).toBe(100);
  });

  it("scrolls backwards on wheel up", () => {
    expect(tabWheelDelta({ ...base, deltaY: -100, scrollLeft: 200 })).toBe(-100);
  });

  it("returns 0 when already at the left edge", () => {
    expect(tabWheelDelta({ ...base, deltaY: -100, scrollLeft: 0 })).toBe(0);
  });

  it("returns 0 when already at the right edge", () => {
    expect(tabWheelDelta({ ...base, deltaY: 100, scrollLeft: 300 })).toBe(0);
  });

  it("clamps the delta to the remaining scroll distance", () => {
    expect(tabWheelDelta({ ...base, deltaY: 100, scrollLeft: 250 })).toBe(50);
  });

  it("lets native horizontal gestures through", () => {
    expect(tabWheelDelta({ ...base, deltaX: 40, deltaY: 10 })).toBe(0);
  });

  it("normalizes line mode to pixels", () => {
    expect(tabWheelDelta({ ...base, deltaY: 3, deltaMode: 1, scrollLeft: 100 })).toBe(48);
  });

  it("normalizes page mode to one viewport width", () => {
    expect(tabWheelDelta({ ...base, deltaY: 1, deltaMode: 2, scrollLeft: 0 })).toBe(300);
  });

  it("returns 0 for a zero or invalid delta", () => {
    expect(tabWheelDelta({ ...base, deltaY: 0 })).toBe(0);
    expect(tabWheelDelta({ ...base, deltaY: Number.NaN })).toBe(0);
  });
});
