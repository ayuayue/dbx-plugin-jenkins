export type UiLocale = "zh" | "en";

const zh: Record<string, string> = {
  unauthorized: "认证失败，请检查用户名与 API Token",
  forbidden: "权限不足，或 CSRF crumb 被拒绝",
  not_found: "Jenkins 地址、上下文路径或资源不存在",
  timeout: "连接 Jenkins 超时",
  network: "无法访问 Jenkins，请检查网络与证书",
  read_only: "当前连接处于只读模式",
  invalid_config: "连接配置不完整，请检查地址与端口",
  invalid_params: "请求参数不完整",
  not_connected: "连接尚未建立，请断开后重新连接",
  method_not_found: "该功能尚未实现",
  server_error: "Jenkins 服务端错误",
  rejected: "请求被 Jenkins 拒绝",
  stages_unavailable: "阶段数据不可用：该任务不是 Pipeline，或未安装 Pipeline REST API 插件",
};

const en: Record<string, string> = {
  unauthorized: "Authentication failed. Check the username and API token.",
  forbidden: "Permission denied, or the CSRF crumb was rejected.",
  not_found: "Jenkins URL, context path, or resource was not found.",
  timeout: "Connecting to Jenkins timed out.",
  network: "Cannot reach Jenkins. Check the network and TLS certificate.",
  read_only: "This connection is in read-only mode.",
  invalid_config: "Incomplete connection settings. Check the host and port.",
  invalid_params: "Missing request parameters.",
  not_connected: "The connection is not established. Disconnect and reconnect.",
  method_not_found: "This feature is not implemented yet.",
  server_error: "Jenkins server error.",
  rejected: "Jenkins rejected the request",
  stages_unavailable: "Stage data unavailable. This is not a Pipeline job, or the Pipeline REST API plugin is missing.",
};

/** RPC 方法前缀，形如 `jenkins/createJob: `；本地化需要先跳过它。 */
const methodPrefix = /^[a-zA-Z][a-zA-Z0-9_]*(?:\/[a-zA-Z][a-zA-Z0-9_]*)+\s*:\s*/;
const kindPrefix = /^([a-z][a-z0-9_]*)\s*:\s*/;

/** 这些错误种类的原因就在 detail 里，拼在本地化前缀后比固定文案有用。 */
const REASON_KINDS = new Set(["rejected", "invalid_config"]);

export interface SplitMessage {
  kind: string | null;
  detail: string;
}

/** 把 `jenkins/createJob: rejected: 原因` 拆成 kind 与原因。 */
export function splitMessage(message: string): SplitMessage {
  const body = (message ?? "").replace(methodPrefix, "");
  const match = kindPrefix.exec(body);
  if (!match) return { kind: null, detail: body };
  return { kind: match[1], detail: body.slice(match[0].length).trim() };
}

/** 从 "<kind>: <detail>" 形式的错误消息里取出稳定错误码。 */
export function errorKind(message: string): string | null {
  return splitMessage(message).kind;
}

/** 把后端错误消息转成当前语言的提示；未知错误码保留原始细节。 */
export function errorText(message: string, locale: UiLocale = "zh"): string {
  if (!message) return "";
  const { kind, detail } = splitMessage(message);
  if (!kind) return message;

  const table = locale === "en" ? en : zh;
  const base = table[kind];
  if (!base) return detail;

  if (REASON_KINDS.has(kind) && detail) {
    return `${base}${locale === "en" ? ": " : "："}${detail}`;
  }
  return base;
}
