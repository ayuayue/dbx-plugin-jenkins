<script setup lang="ts">
import { computed } from "vue";
import { t } from "../i18n";
import { formatDuration, formatTimestamp } from "../lib/format";
import { buildProgress, type BuildProgress } from "../lib/progress";
import type { RefreshUi } from "../lib/refreshStatus";
import { statusMeta } from "../lib/status";
import type { BuildSummary, JobDetail, StageInfo } from "../types";
import BuildProgressBar from "./BuildProgressBar.vue";
import RefreshBadge from "./RefreshBadge.vue";

const props = defineProps<{
  job: JobDetail;
  builds: BuildSummary[];
  loadingBuilds: boolean;
  readOnly: boolean;
  isPipeline: boolean;
  locale: "zh" | "en";
  refresh: RefreshUi;
  changedBuilds: number[];
  /** 正在构建的那一次构建的阶段快照；有就按阶段步进，没有则回退时长估算。 */
  stageSnapshot: { number: number; stages: StageInfo[] } | null;
}>();

const emit = defineEmits<{
  build: [];
  buildWithParams: [];
  stop: [];
  openBuild: [number: number];
  openStages: [];
  refresh: [];
  editJob: [];
  deleteJob: [];
}>();

const parameters = computed(() => props.job.parameters ?? []);
const lastBuild = computed(() => props.job.lastBuild ?? null);
const buildLabel = computed(() =>
  parameters.value.length > 0 ? t("buildWithParams") : t("buildNow"),
);

/** 每行构建的进度；已完成的构建为 null，列表显示「—」。 */
const progressByNumber = computed(() => {
  const snapshot = props.stageSnapshot;
  const map = new Map<number, BuildProgress | null>();
  for (const build of props.builds) {
    const stages = snapshot?.number === build.number ? snapshot.stages : null;
    map.set(build.number, buildProgress(build, stages));
  }
  return map;
});

function triggerBuild(): void {
  if (parameters.value.length > 0) {
    emit("buildWithParams");
  } else {
    emit("build");
  }
}
</script>

<template>
  <section class="job-view">
    <header class="job-head">
      <div class="title-block">
        <div class="title-row">
          <span class="status-dot" :data-tone="statusMeta(job.status).tone" />
          <h1>{{ job.name }}</h1>
          <span class="status-pill" :data-tone="statusMeta(job.status).tone">
            {{ t(statusMeta(job.status).key) }}
          </span>
        </div>
        <p class="full-name">{{ job.fullName }}</p>
        <p v-if="job.description" class="description">{{ job.description }}</p>
      </div>

      <div class="actions">
        <template v-if="!readOnly">
          <button type="button" class="primary" :disabled="!job.buildable" @click="triggerBuild">
            {{ buildLabel }}
          </button>
          <button
            v-if="lastBuild?.building"
            type="button"
            class="danger"
            @click="emit('stop')"
          >
            {{ t("stopBuild") }}
          </button>
        </template>
        <span v-else class="read-only">{{ t("readOnlyBadge") }}</span>
        <button
          v-if="isPipeline"
          type="button"
          class="ghost stage-button"
          @click="emit('openStages')"
        >
          {{ t("stageView") }}
        </button>
        <button v-if="!readOnly" type="button" class="ghost" @click="emit('editJob')">
          {{ t("editJob") }}
        </button>
        <button
          v-if="!readOnly"
          type="button"
          class="ghost danger-text"
          @click="emit('deleteJob')"
        >
          {{ t("deleteJob") }}
        </button>
        <button
          type="button"
          class="ghost"
          :class="{ done: refresh.refreshed }"
          :disabled="refresh.refreshing"
          @click="emit('refresh')"
        >
          {{
            refresh.refreshing
              ? t("refreshing")
              : refresh.refreshed
                ? t("refreshed")
                : t("refresh")
          }}
        </button>
      </div>
    </header>

    <div class="meta-grid">
      <div class="meta">
        <span class="meta-label">{{ t("lastBuild") }}</span>
        <strong>{{ lastBuild ? `#${lastBuild.number}` : "—" }}</strong>
      </div>
      <div class="meta">
        <span class="meta-label">{{ t("status") }}</span>
        <strong>{{ lastBuild ? t(statusMeta(lastBuild.building ? "building" : job.status).key) : "—" }}</strong>
      </div>
      <div class="meta">
        <span class="meta-label">{{ t("startedAt") }}</span>
        <strong>{{ lastBuild ? formatTimestamp(lastBuild.timestamp, locale) : "—" }}</strong>
      </div>
      <div class="meta">
        <span class="meta-label">{{ t("duration") }}</span>
        <strong>{{ lastBuild ? formatDuration(lastBuild.duration) : "—" }}</strong>
      </div>
    </div>

    <section class="panel">
      <div class="panel-head">
        <h2>{{ t("parameters") }}</h2>
        <span class="count">{{ parameters.length }}</span>
      </div>
      <p v-if="!parameters.length" class="empty">{{ t("noParameters") }}</p>
      <ul v-else class="params">
        <li v-for="parameter in parameters" :key="parameter.name">
          <div class="param-line">
            <code>{{ parameter.name }}</code>
            <span class="param-type">{{ parameter.type.replace("ParameterDefinition", "") || "String" }}</span>
          </div>
          <p v-if="parameter.description" class="param-desc">{{ parameter.description }}</p>
          <p v-if="parameter.choices?.length" class="param-default">
            {{ parameter.choices.join(" / ") }}
          </p>
          <p v-else-if="parameter.defaultValue" class="param-default">
            = {{ parameter.defaultValue }}
          </p>
        </li>
      </ul>
    </section>

    <section class="panel">
      <div class="panel-head">
        <h2>{{ t("builds") }}</h2>
        <span class="panel-meta">
          <RefreshBadge
            :mode="refresh.mode"
            :paused="refresh.paused"
            :in-flight="refresh.inFlight"
            :last-at="refresh.lastAt"
            :interval-ms="refresh.intervalMs"
          />
          <span class="count">{{ builds.length }}</span>
        </span>
      </div>
      <p v-if="loadingBuilds" class="empty">{{ t("loading") }}</p>
      <p v-else-if="!builds.length" class="empty">{{ t("noBuilds") }}</p>
      <table v-else class="builds">
        <thead>
          <tr>
            <th>#</th>
            <th>{{ t("status") }}</th>
            <th>{{ t("progress") }}</th>
            <th>{{ t("startedAt") }}</th>
            <th>{{ t("duration") }}</th>
            <th>{{ t("cause") }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="build in builds"
            :key="build.number"
            :class="{ changed: changedBuilds.includes(build.number) }"
            @click="emit('openBuild', build.number)"
          >
            <td class="number">#{{ build.number }}</td>
            <td>
              <span class="status-pill small" :data-tone="statusMeta(build.status).tone">
                {{ t(statusMeta(build.status).key) }}
              </span>
            </td>
            <td class="progress-cell">
              <BuildProgressBar :progress="progressByNumber.get(build.number) ?? null" />
            </td>
            <td>{{ formatTimestamp(build.timestamp, locale) }}</td>
            <td>{{ formatDuration(build.duration) }}</td>
            <td class="cause">{{ build.causes?.[0]?.shortDescription ?? "—" }}</td>
          </tr>
        </tbody>
      </table>
    </section>
  </section>
</template>

<style scoped>
.job-view {
  flex: 1;
  min-width: 0;
  overflow: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.job-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.title-row {
  display: flex;
  align-items: center;
  gap: 9px;
}

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--muted);
}

.status-dot[data-tone="success"] {
  background: var(--success);
}

.status-dot[data-tone="danger"] {
  background: var(--danger);
}

.status-dot[data-tone="warning"] {
  background: #d99b1a;
}

.status-dot[data-tone="info"] {
  background: var(--accent);
  box-shadow: 0 0 0 4px var(--accent-soft);
}

h1 {
  margin: 0;
  font-size: 20px;
}

.full-name {
  margin: 4px 0 0 19px;
  color: var(--muted);
  font-size: 12px;
}

.description {
  margin: 10px 0 0 19px;
  max-width: 760px;
  color: var(--muted);
  line-height: 1.65;
}

.actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: none;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.stage-button {
  border-color: var(--accent);
  color: var(--accent);
}

.primary,
.danger,
.ghost {
  border-radius: var(--radius-sm);
  padding: 8px 13px;
  font-weight: 600;
}

.primary {
  border: 0;
  background: var(--accent);
  color: #fff;
}

.primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.danger {
  border: 1px solid color-mix(in srgb, var(--danger) 45%, transparent);
  background: transparent;
  color: var(--danger);
}

.ghost {
  border: 1px solid var(--border);
  background: transparent;
  color: var(--text);
}

.ghost:hover {
  border-color: var(--accent);
  color: var(--accent);
}

.ghost:disabled {
  opacity: 0.55;
  cursor: default;
  border-color: var(--border);
  color: var(--muted);
}

.ghost.done {
  border-color: var(--success);
  color: var(--success);
}

.ghost.danger-text {
  color: var(--danger);
}

.ghost.danger-text:hover {
  border-color: var(--danger);
  color: var(--danger);
}

.read-only {
  padding: 5px 10px;
  border-radius: 999px;
  background: var(--accent-soft);
  color: var(--accent);
  font-size: 12px;
  font-weight: 600;
}

.status-pill {
  padding: 3px 9px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  background: var(--panel-soft);
  color: var(--muted);
}

.status-pill[data-tone="success"] {
  background: color-mix(in srgb, var(--success) 14%, transparent);
  color: var(--success);
}

.status-pill[data-tone="danger"] {
  background: color-mix(in srgb, var(--danger) 14%, transparent);
  color: var(--danger);
}

.status-pill[data-tone="warning"] {
  background: #d99b1a1f;
  color: #b47d0d;
}

.status-pill[data-tone="info"] {
  background: var(--accent-soft);
  color: var(--accent);
}

.status-pill.small {
  font-size: 10px;
}

.meta-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 12px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  background: var(--panel);
}

.meta-label {
  color: var(--muted);
  font-size: 11px;
}

.meta strong {
  font-size: 13px;
  font-variant-numeric: tabular-nums;
}

.panel {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 16px;
}

.panel-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 10px;
}

h2 {
  margin: 0;
  font-size: 14px;
}

.count {
  color: var(--muted);
  font-size: 12px;
}

.panel-meta {
  display: flex;
  align-items: center;
  gap: 10px;
}

.empty {
  margin: 8px 0;
  color: var(--muted);
  font-size: 12px;
}

.params {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 10px;
}

.params li {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 10px 12px;
  background: var(--panel-soft);
}

.param-line {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.param-line code {
  font-size: 12px;
  font-weight: 600;
}

.param-type {
  color: var(--muted);
  font-size: 10px;
}

.param-desc,
.param-default {
  margin: 6px 0 0;
  color: var(--muted);
  font-size: 11px;
  line-height: 1.5;
}

.builds {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}

.builds th {
  text-align: left;
  color: var(--muted);
  font-weight: 600;
  padding: 7px 8px;
  border-bottom: 1px solid var(--border);
}

.builds td {
  padding: 8px;
  border-bottom: 1px solid var(--border);
}

.builds tbody tr {
  cursor: pointer;
  transition: background-color 0.4s ease;
}

.builds tbody tr:hover {
  background: var(--panel-soft);
}

/* 新增或状态刚翻转的构建：左侧绿条短暂标注，之后自动淡出 */
.builds tbody tr.changed {
  background: color-mix(in srgb, var(--success) 9%, transparent);
  box-shadow: inset 2px 0 0 var(--success);
}

.builds tbody tr.changed .status-pill {
  animation: pill-flash 1.2s ease-out;
}

@keyframes pill-flash {
  0% {
    transform: scale(1);
  }
  35% {
    transform: scale(1.08);
  }
  100% {
    transform: scale(1);
  }
}

.number {
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}

.progress-cell {
  width: 132px;
}

.cause {
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: var(--muted);
}
</style>
