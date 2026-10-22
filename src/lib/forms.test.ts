import { describe, expect, it } from "vitest";
import {
  buildCreateJobPayload,
  buildCreateViewPayload,
  buildJobUpdateFields,
  buildViewUpdateFields,
  deleteConfirmationMatches,
  splitJobNames,
  validateName,
} from "./forms";

describe("validateName", () => {
  it("accepts plain, chinese and dotted names", () => {
    expect(validateName("demo")).toBeNull();
    expect(validateName("发布任务")).toBeNull();
    expect(validateName("release-1.0")).toBeNull();
    expect(validateName("my job")).toBeNull();
  });

  it("rejects empty input", () => {
    expect(validateName("")).toBe("nameRequired");
    expect(validateName("   ")).toBe("nameRequired");
  });

  it("rejects path separators and traversal", () => {
    expect(validateName("a/b")).toBe("nameInvalid");
    expect(validateName("a\\b")).toBe("nameInvalid");
    expect(validateName("..")).toBe("nameInvalid");
    expect(validateName("a..b")).toBe("nameInvalid");
    expect(validateName("a?b")).toBe("nameInvalid");
    expect(validateName("a#b")).toBe("nameInvalid");
    expect(validateName("a%b")).toBe("nameInvalid");
  });

  it("rejects names longer than 255 characters", () => {
    expect(validateName("a".repeat(255))).toBeNull();
    expect(validateName("a".repeat(256))).toBe("nameInvalid");
  });
});

describe("buildCreateJobPayload", () => {
  it("trims every text field and keeps the folder", () => {
    expect(
      buildCreateJobPayload({
        folder: " team ",
        name: " demo ",
        kind: "pipeline",
        description: " nightly ",
        script: " echo hi ",
      }),
    ).toEqual({
      folder: "team",
      name: "demo",
      kind: "pipeline",
      description: "nightly",
      script: "echo hi",
    });
  });

  it("drops the script for free-style jobs", () => {
    const payload = buildCreateJobPayload({
      folder: "",
      name: "demo",
      kind: "freestyle",
      description: "",
      script: "echo hi",
    });
    expect(payload.script).toBe("");
  });
});

describe("buildCreateViewPayload", () => {
  it("trims text, dedupes jobs and normalizes regex", () => {
    expect(
      buildCreateViewPayload({
        name: " Builds ",
        description: " main jobs ",
        jobs: ["demo", " demo ", "api"],
        includeRegex: " ^rel ",
        recurse: true,
      }),
    ).toEqual({
      name: "Builds",
      description: "main jobs",
      jobs: ["demo", "api"],
      includeRegex: "^rel",
      recurse: true,
    });
  });
});

describe("buildJobUpdateFields", () => {
  const original = {
    title: "demo",
    description: "old",
    disabled: false,
    script: "echo old",
  };

  it("returns only changed fields", () => {
    expect(
      buildJobUpdateFields({ ...original, description: "new" }, original),
    ).toEqual({ description: "new" });
  });

  it("returns an empty object when nothing changed", () => {
    expect(buildJobUpdateFields({ ...original }, original)).toEqual({});
  });

  it("ignores an empty title (no rename)", () => {
    expect(
      buildJobUpdateFields({ ...original, title: "  " }, original),
    ).toEqual({});
  });

  it("includes disabled and script changes", () => {
    expect(
      buildJobUpdateFields({ ...original, disabled: true, script: "echo new" }, original),
    ).toEqual({ disabled: true, script: "echo new" });
  });
});

describe("buildViewUpdateFields", () => {
  const original = {
    name: "Builds",
    description: "old",
    jobNames: ["demo", "api"],
    includeRegex: "",
    recurse: false,
  };

  it("returns only changed fields", () => {
    expect(
      buildViewUpdateFields({ ...original, recurse: true }, original),
    ).toEqual({ recurse: true });
  });

  it("compares job lists as sets", () => {
    expect(
      buildViewUpdateFields({ ...original, jobNames: ["api", "demo"] }, original),
    ).toEqual({});
  });

  it("detects added jobs", () => {
    expect(
      buildViewUpdateFields({ ...original, jobNames: ["demo", "api", "web"] }, original),
    ).toEqual({ jobNames: ["demo", "api", "web"] });
  });

  it("detects a renamed view", () => {
    expect(
      buildViewUpdateFields({ ...original, name: "Builds 2" }, original),
    ).toEqual({ name: "Builds 2" });
  });

  it("trims the new name before comparing", () => {
    expect(
      buildViewUpdateFields({ ...original, name: "  Builds 2  " }, original),
    ).toEqual({ name: "Builds 2" });
  });

  it("ignores whitespace-only differences", () => {
    expect(
      buildViewUpdateFields({ ...original, name: "  Builds " }, original),
    ).toEqual({});
  });

  it("ignores an empty name (no rename)", () => {
    expect(buildViewUpdateFields({ ...original, name: "   " }, original)).toEqual({});
  });
});

describe("splitJobNames", () => {
  it("splits on comma and newline, trims and dedupes", () => {
    expect(splitJobNames(" demo, api\ndemo ,\n web ")).toEqual([
      "demo",
      "api",
      "web",
    ]);
  });

  it("returns an empty array for blank input", () => {
    expect(splitJobNames("  \n ")).toEqual([]);
  });
});

describe("deleteConfirmationMatches", () => {
  it("requires an exact trimmed match", () => {
    expect(deleteConfirmationMatches("demo", "demo")).toBe(true);
    expect(deleteConfirmationMatches(" demo ", "demo")).toBe(true);
    expect(deleteConfirmationMatches("Demo", "demo")).toBe(false);
    expect(deleteConfirmationMatches("", "demo")).toBe(false);
  });
});
