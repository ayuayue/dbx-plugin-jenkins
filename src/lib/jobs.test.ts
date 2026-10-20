import { describe, expect, it } from "vitest";
import { groupJobs } from "./jobs";
import type { JobSummary } from "../types";

function job(name: string, isFolder = false): JobSummary {
  return {
    name,
    fullName: name,
    url: "",
    color: "blue",
    isFolder,
    status: "success",
    building: false,
  };
}

describe("groupJobs", () => {
  it("returns empty groups for an empty list", () => {
    expect(groupJobs([])).toEqual({ folders: [], jobs: [] });
  });

  it("splits folders and jobs without changing order", () => {
    const result = groupJobs([
      job("team", true),
      job("demo"),
      job("ops", true),
      job("api"),
    ]);
    expect(result.folders.map((item) => item.name)).toEqual(["team", "ops"]);
    expect(result.jobs.map((item) => item.name)).toEqual(["demo", "api"]);
  });

  it("handles a list with only jobs", () => {
    const result = groupJobs([job("demo"), job("api")]);
    expect(result.folders).toEqual([]);
    expect(result.jobs).toHaveLength(2);
  });
});
