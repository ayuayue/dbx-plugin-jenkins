import { describe, expect, it } from "vitest";
import { buildProgress, stageProgress } from "./progress";

describe("buildProgress", () => {
  it("returns null for finished builds", () => {
    expect(
      buildProgress({ building: false, duration: 30_000, estimatedDuration: 60_000 }),
    ).toBeNull();
  });

  it("computes percent from elapsed duration over estimate", () => {
    expect(
      buildProgress({ building: true, duration: 15_000, estimatedDuration: 60_000 }),
    ).toEqual({ percent: 25, indeterminate: false });
  });

  it("caps a still-running build below 100%", () => {
    expect(
      buildProgress({ building: true, duration: 90_000, estimatedDuration: 60_000 }),
    ).toEqual({ percent: 99, indeterminate: false });
  });

  it("falls back to indeterminate when the estimate is missing", () => {
    expect(
      buildProgress({ building: true, duration: 5_000, estimatedDuration: 0 }),
    ).toEqual({ percent: 0, indeterminate: true });
  });

  it("clamps a just-started build to zero", () => {
    expect(
      buildProgress({ building: true, duration: 0, estimatedDuration: 60_000 }),
    ).toEqual({ percent: 0, indeterminate: false });
  });
});

describe("stageProgress", () => {
  it("counts a running stage as half a step", () => {
    expect(
      stageProgress([
        { status: "success" },
        { status: "in_progress" },
        { status: "not_executed" },
      ]),
    ).toEqual({ percent: 50, indeterminate: false });
  });

  it("treats skipped stages as done", () => {
    expect(stageProgress([{ status: "success" }, { status: "skipped" }])).toEqual({
      percent: 99,
      indeterminate: false,
    });
  });

  it("caps at 99 while the build is still running", () => {
    expect(stageProgress([{ status: "success" }, { status: "success" }])).toEqual({
      percent: 99,
      indeterminate: false,
    });
  });

  it("counts a paused stage as running", () => {
    expect(stageProgress([{ status: "success" }, { status: "paused" }])).toEqual({
      percent: 75,
      indeterminate: false,
    });
  });

  it("returns null without stages", () => {
    expect(stageProgress([])).toBeNull();
  });
});

describe("buildProgress with stage data", () => {
  it("prefers stage steps over the duration estimate", () => {
    expect(
      buildProgress({ building: true, duration: 30_000, estimatedDuration: 60_000 }, [
        { status: "success" },
        { status: "in_progress" },
        { status: "not_executed" },
      ]),
    ).toEqual({ percent: 50, indeterminate: false });
  });

  it("falls back to the estimate when stage data is empty", () => {
    expect(
      buildProgress({ building: true, duration: 15_000, estimatedDuration: 60_000 }, []),
    ).toEqual({ percent: 25, indeterminate: false });
  });

  it("still returns null for finished builds even with stages", () => {
    expect(
      buildProgress({ building: false, duration: 60_000, estimatedDuration: 60_000 }, [
        { status: "success" },
      ]),
    ).toBeNull();
  });
});
