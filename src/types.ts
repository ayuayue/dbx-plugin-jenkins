export type JenkinsStatus =
  | "success"
  | "failure"
  | "unstable"
  | "building"
  | "aborted"
  | "disabled"
  | "not_built"
  | "unknown";

export interface ViewInfo {
  name: string;
  description: string;
  url: string;
}

export interface BuildRef {
  number: number;
  result: string;
  timestamp: number;
  duration: number;
  building: boolean;
}

export interface JobSummary {
  name: string;
  fullName: string;
  url: string;
  color: string;
  isFolder: boolean;
  status: JenkinsStatus;
  building: boolean;
  lastBuild?: BuildRef | null;
}

export interface ParameterDefinition {
  name: string;
  type: string;
  description: string;
  defaultValue: string;
  choices?: string[] | null;
}

export interface JobDetail {
  name: string;
  fullName: string;
  description: string;
  url: string;
  color: string;
  class?: string;
  buildable: boolean;
  inQueue: boolean;
  lastBuild?: BuildRef | null;
  lastSuccessfulBuild?: BuildRef | null;
  lastFailedBuild?: BuildRef | null;
  parameters: ParameterDefinition[] | null;
  status: JenkinsStatus;
}

export interface Cause {
  shortDescription: string;
  userId?: string;
  userName?: string;
}

export interface BuildSummary {
  number: number;
  result: string;
  building: boolean;
  timestamp: number;
  duration: number;
  displayName: string;
  causes: Cause[] | null;
  status: JenkinsStatus;
}

export interface ChangeSetItem {
  commitId: string;
  message: string;
  author: string;
}

export interface Artifact {
  fileName: string;
  relativePath: string;
}

export interface BuildDetail extends BuildSummary {
  estimatedDuration: number;
  fullDisplayName: string;
  url: string;
  changeSet: ChangeSetItem[] | null;
  artifacts: Artifact[] | null;
}

export interface ConsoleChunk {
  text: string;
  nextStart: number;
  moreData: boolean;
}

export interface StageInfo {
  id: string;
  name: string;
  status: string;
  startTimeMillis: number;
  durationMillis: number;
  pauseDurationMillis: number;
}

export interface StageBuildInfo {
  number: number;
  result: string;
  status: string;
  timestamp: number;
  duration: number;
  commitCount: number;
  stages: StageInfo[] | null;
}

export interface StageAverageInfo {
  name: string;
  duration: number;
}

export interface StageMatrix {
  job: string;
  stageNames: string[];
  builds: StageBuildInfo[];
  averages: StageAverageInfo[];
  averageRun: number;
}

export interface QueueItem {
  id: number;
  jobName: string;
  jobUrl: string;
  why: string;
  stuck: boolean;
  blocked: boolean;
  buildableIn: number;
  inQueueSince: number;
  cause: string;
}

export interface NodeInfo {
  name: string;
  offline: boolean;
  temporarilyOffline: boolean;
  numExecutors: number;
  idle: number;
  busy: number;
  labels: string[] | null;
  offlineReason: string;
}

export interface Overview {
  version: string;
  user: string;
  mode: string;
  nodeDescription: string;
  quietingDown: boolean;
  queueLength: number;
  nodesOnline: number;
  nodesTotal: number;
  executorsBusy: number;
  executorsTotal: number;
}

export interface TriggerResult {
  success: boolean;
  queueId: number;
}

export interface WhoAmIResponse {
  success: boolean;
  readOnly: boolean;
  message: string;
  user: {
    name: string;
    authenticated: boolean;
    authorities?: string[] | null;
    mode?: string;
    version?: string;
  };
}

export type JobKind = "freestyle" | "pipeline";

export interface JobFields {
  title: string;
  kind: JobKind;
  description: string;
  disabled: boolean;
  script: string;
}

export interface JobConfigResponse {
  xml: string;
  fields: JobFields;
}

export interface ListViewFields {
  name: string;
  description: string;
  jobNames: string[] | null;
  includeRegex: string;
  recurse: boolean;
}

export interface ViewConfigResponse {
  xml: string;
  fields: ListViewFields | null;
}

export interface JobUpdateFields {
  title?: string;
  description?: string;
  disabled?: boolean;
  script?: string;
}

export interface ViewUpdateFields {
  description?: string;
  jobNames?: string[];
  includeRegex?: string;
  recurse?: boolean;
}

export interface CreateJobPayload {
  folder: string;
  name: string;
  kind: JobKind;
  description: string;
  script: string;
}

export interface CreateViewPayload {
  name: string;
  description: string;
  jobs: string[];
  includeRegex: string;
  recurse: boolean;
}

export interface MutationResult {
  success: boolean;
  fullName?: string;
}
