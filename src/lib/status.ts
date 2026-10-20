import type { MessageKey } from "../i18n";

export type StatusTone = "success" | "danger" | "warning" | "info" | "muted";

export interface StatusMeta {
  key: MessageKey;
  tone: StatusTone;
}

const statuses: Record<string, StatusMeta> = {
  success: { key: "statusSuccess", tone: "success" },
  failure: { key: "statusFailure", tone: "danger" },
  unstable: { key: "statusUnstable", tone: "warning" },
  building: { key: "statusBuilding", tone: "info" },
  aborted: { key: "statusAborted", tone: "muted" },
  disabled: { key: "statusDisabled", tone: "muted" },
  not_built: { key: "statusNotBuilt", tone: "muted" },
  unknown: { key: "statusUnknown", tone: "muted" },
};

export function statusMeta(status: string): StatusMeta {
  return statuses[status] ?? statuses.unknown;
}
