import { describe, expect, it } from "vitest";
import { statusMeta } from "./status";

describe("statusMeta", () => {
  it("maps Jenkins statuses to tones", () => {
    expect(statusMeta("success").tone).toBe("success");
    expect(statusMeta("failure").tone).toBe("danger");
    expect(statusMeta("unstable").tone).toBe("warning");
    expect(statusMeta("building").tone).toBe("info");
    expect(statusMeta("aborted").tone).toBe("muted");
    expect(statusMeta("disabled").tone).toBe("muted");
    expect(statusMeta("not_built").tone).toBe("muted");
  });

  it("falls back to unknown", () => {
    expect(statusMeta("wat").key).toBe("statusUnknown");
    expect(statusMeta("").key).toBe("statusUnknown");
  });
});
