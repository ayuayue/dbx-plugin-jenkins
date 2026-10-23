import { describe, expect, it } from "vitest";
import {
  createCancelRequest,
  createStopRequest,
  isStopTarget,
} from "./confirmRequest";

describe("createStopRequest", () => {
  it("captures the job full name and build number", () => {
    expect(createStopRequest("folder/app", 42)).toEqual({
      kind: "stop",
      job: "folder/app",
      number: 42,
    });
  });

  it("trims the job name", () => {
    expect(createStopRequest("  app  ", 7)).toEqual({
      kind: "stop",
      job: "app",
      number: 7,
    });
  });

  it("rejects a missing job name", () => {
    expect(createStopRequest("", 42)).toBeNull();
    expect(createStopRequest("   ", 42)).toBeNull();
    expect(createStopRequest(null, 42)).toBeNull();
    expect(createStopRequest(undefined, 42)).toBeNull();
  });

  it("rejects invalid build numbers", () => {
    expect(createStopRequest("app", 0)).toBeNull();
    expect(createStopRequest("app", -3)).toBeNull();
    expect(createStopRequest("app", 1.5)).toBeNull();
    expect(createStopRequest("app", Number.NaN)).toBeNull();
    expect(createStopRequest("app", null)).toBeNull();
    expect(createStopRequest("app", undefined)).toBeNull();
  });
});

describe("createCancelRequest", () => {
  it("captures the queue id and job name", () => {
    expect(createCancelRequest(123, "folder/app")).toEqual({
      kind: "cancel",
      id: "123",
      jobName: "folder/app",
    });
  });

  it("falls back to an empty job name", () => {
    expect(createCancelRequest(5)).toEqual({
      kind: "cancel",
      id: "5",
      jobName: "",
    });
    expect(createCancelRequest(5, null)).toEqual({
      kind: "cancel",
      id: "5",
      jobName: "",
    });
  });

  it("rejects invalid queue ids", () => {
    expect(createCancelRequest(0)).toBeNull();
    expect(createCancelRequest(-1)).toBeNull();
    expect(createCancelRequest(1.2)).toBeNull();
    expect(createCancelRequest(Number.NaN)).toBeNull();
    expect(createCancelRequest(null)).toBeNull();
    expect(createCancelRequest(undefined)).toBeNull();
  });
});

describe("isStopTarget", () => {
  it("matches only the same job", () => {
    const request = createStopRequest("app", 3);
    expect(isStopTarget(request, "app")).toBe(true);
    expect(isStopTarget(request, "other")).toBe(false);
  });

  it("never matches an empty or non-stop request", () => {
    expect(isStopTarget(null, "app")).toBe(false);
    expect(isStopTarget(createCancelRequest(9), "app")).toBe(false);
    expect(isStopTarget(createStopRequest("app", 3), undefined)).toBe(false);
  });
});
