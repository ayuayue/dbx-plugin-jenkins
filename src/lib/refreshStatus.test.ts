import { describe, expect, it } from "vitest";
import type { BuildSummary } from "../types";
import { diffBuildRows, formatClockTime, refreshBadge } from "./refreshStatus";

function build(
  number: number,
  status: BuildSummary["status"] = "success",
): BuildSummary {
  return {
    number,
    result: status.toUpperCase(),
    building: status === "building",
    timestamp: 1700000000000,
    duration: 1000,
    estimatedDuration: 0,
    displayName: `#${number}`,
    causes: null,
    status,
  };
}

describe("refreshBadge", () => {
  it("stays idle without a polling mode", () => {
    expect(refreshBadge(null, false, false)).toBe("idle");
  });

  it("marks an active poller", () => {
    expect(refreshBadge("builds", false, false)).toBe("active");
  });

  it("marks the run in flight", () => {
    expect(refreshBadge("builds", false, true)).toBe("busy");
  });

  it("lets the paused state win over the in-flight state", () => {
    expect(refreshBadge("builds", true, true)).toBe("paused");
  });

  it("stays idle when paused but nothing needs polling", () => {
    expect(refreshBadge(null, true, true)).toBe("idle");
  });
});

describe("formatClockTime", () => {
  it("returns an empty string without a timestamp", () => {
    expect(formatClockTime(null)).toBe("");
    expect(formatClockTime(undefined)).toBe("");
    expect(formatClockTime(0)).toBe("");
  });

  it("pads hours, minutes and seconds", () => {
    const at = new Date(2026, 9, 22, 9, 5, 3).getTime();
    expect(formatClockTime(at)).toBe("09:05:03");
  });

  it("handles late hours without wrapping", () => {
    const at = new Date(2026, 9, 22, 23, 59, 58).getTime();
    expect(formatClockTime(at)).toBe("23:59:58");
  });
});

describe("diffBuildRows", () => {
  it("returns nothing on the first load", () => {
    expect(diffBuildRows(null, [build(1)])).toEqual([]);
    expect(diffBuildRows(undefined, [build(1)])).toEqual([]);
  });

  it("returns nothing when nothing changed", () => {
    expect(diffBuildRows([build(2), build(1)], [build(2), build(1)])).toEqual([]);
  });

  it("detects a new build", () => {
    expect(diffBuildRows([build(1)], [build(2), build(1)])).toEqual([2]);
  });

  it("detects a status flip on an existing build", () => {
    expect(
      diffBuildRows([build(1, "building"), build(2)], [build(1, "success"), build(2)]),
    ).toEqual([1]);
  });

  it("highlights the first build of an empty job", () => {
    expect(diffBuildRows([], [build(1)])).toEqual([1]);
  });

  it("ignores an empty next list", () => {
    expect(diffBuildRows([build(1)], [])).toEqual([]);
    expect(diffBuildRows([build(1)], null)).toEqual([]);
  });
});
