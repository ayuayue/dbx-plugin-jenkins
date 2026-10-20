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
  stages_unavailable: "Stage data unavailable. This is not a Pipeline job, or the Pipeline REST API plugin is missing.",
};

const prefix = /^([a-z][a-z0-9_]*):\s*/;

/** 从 "<kind>: <detail>" 形式的错误消息里取出稳定错误码。 */
export function errorKind(message: string): string | null {
  const match = prefix.exec(message ?? "");
  return match ? match[1] : null;
}

/** 把后端错误消息转成当前语言的提示；未知错误码保留原始细节。 */
export function errorText(message: string, locale: UiLocale = "zh"): string {
  if (!message) return "";
  const kind = errorKind(message);
  if (!kind) return message;

  const table = locale === "en" ? en : zh;
  if (table[kind]) return table[kind];
  return message.replace(prefix, "");
}
