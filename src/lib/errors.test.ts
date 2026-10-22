import { describe, expect, it } from "vitest";
import { errorKind, errorText } from "./errors";

describe("errorKind", () => {
  it("reads the stable kind prefix", () => {
    expect(errorKind("unauthorized: 401 Unauthorized")).toBe("unauthorized");
    expect(errorKind("not_connected: reconnect")).toBe("not_connected");
  });

  it("returns null when there is no prefix", () => {
    expect(errorKind("network down")).toBeNull();
    expect(errorKind("")).toBeNull();
  });

  it("reads the kind after an RPC method prefix", () => {
    expect(errorKind("jenkins/createJob: rejected: boom")).toBe("rejected");
    expect(errorKind("connection/connect: timeout: dial")).toBe("timeout");
    expect(errorKind("mcp/call: forbidden: denied")).toBe("forbidden");
  });
});

describe("errorText", () => {
  it("localizes known kinds in Chinese", () => {
    expect(errorText("unauthorized: 401", "zh")).toBe("认证失败，请检查用户名与 API Token");
    expect(errorText("forbidden: 403", "zh")).toBe("权限不足，或 CSRF crumb 被拒绝");
    expect(errorText("not_found: 404", "zh")).toBe("Jenkins 地址、上下文路径或资源不存在");
    expect(errorText("timeout: deadline", "zh")).toBe("连接 Jenkins 超时");
    expect(errorText("network: dial", "zh")).toBe("无法访问 Jenkins，请检查网络与证书");
    expect(errorText("read_only: blocked", "zh")).toBe("当前连接处于只读模式");
    expect(errorText("invalid_config: host", "zh")).toBe(
      "连接配置不完整，请检查地址与端口：host",
    );
    expect(errorText("not_connected: none", "zh")).toBe("连接尚未建立，请断开后重新连接");
  });

  it("localizes known kinds in English", () => {
    expect(errorText("unauthorized: 401", "en")).toBe(
      "Authentication failed. Check the username and API token.",
    );
    expect(errorText("timeout: deadline", "en")).toBe("Connecting to Jenkins timed out.");
  });

  it("strips an unknown prefix but keeps the detail", () => {
    expect(errorText("internal: boom", "zh")).toBe("boom");
    expect(errorText("weird_kind: detail", "en")).toBe("detail");
  });

  it("keeps plain messages unchanged", () => {
    expect(errorText("plain failure", "zh")).toBe("plain failure");
  });

  it("localizes kinds that carry an RPC method prefix", () => {
    expect(errorText("jenkins/jobs: unauthorized: 401", "zh")).toBe(
      "认证失败，请检查用户名与 API Token",
    );
    expect(errorText("jenkins/trigger: read_only: blocked", "zh")).toBe("当前连接处于只读模式");
  });

  it("shows the Jenkins reason for rejected requests", () => {
    expect(
      errorText("jenkins/createJob: rejected: A job already exists with the name 'demo'", "zh"),
    ).toBe("请求被 Jenkins 拒绝：A job already exists with the name 'demo'");
    expect(errorText("jenkins/createJob: rejected: unsafe character", "en")).toBe(
      "Jenkins rejected the request: unsafe character",
    );
  });

  it("keeps the detail when a config error explains the problem", () => {
    expect(errorText("invalid_config: name must not contain *", "zh")).toContain(
      "name must not contain *",
    );
    expect(errorText("invalid_config: host is empty", "en")).toContain("host is empty");
  });
});
