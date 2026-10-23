<script setup lang="ts">
import { computed, ref } from "vue";
import { t } from "../i18n";
import {
  CHANGE_PREVIEW_LIMIT,
  changeBody,
  changeSummary,
  formatChangeTime,
  hasBody,
  sortChanges,
  visibleChanges,
} from "../lib/changes";
import { formatDuration, formatTimestamp } from "../lib/format";
import type { RefreshUi } from "../lib/refreshStatus";
import { shortCommitId } from "../lib/stage";
import { statusMeta } from "../lib/status";
import type { BuildDetail } from "../types";
import ConsoleView from "./ConsoleView.vue";
import RefreshBadge from "./RefreshBadge.vue";

const props = defineProps<{
  build: BuildDetail;
  jobFullName: string;
  readOnly: boolean;
  locale: "zh" | "en";
  refresh: RefreshUi;
}>();

const emit = defineEmits<{
  back: [];
  stop: [number: number];
  refresh: [];
}>();

/** 变更列表默认折叠，避免几十条提交把控制台挤到页面底部。 */
const expanded = ref(false);
/** 已展开正文的提交号。用 Set 存，3s 轮询换数据时不会把展开状态冲掉。 */
const expandedBodies = ref(new Set<string>());
const copiedId = ref("");

const orderedChanges = computed(() => sortChanges(props.build.changeSet ?? []));
const summary = computed(() => changeSummary(orderedChanges.value));
const rows = computed(() =>
  visibleChanges(orderedChanges.value, expanded.value, CHANGE_PREVIEW_LIMIT),
);
const summaryText = computed(() =>
  t("changeSummary", {
    commits: summary.value.commits,
    authors: summary.value.authors,
    files: summary.value.files,
  }),
);

function toggleBody(commitId: string): void {
  const next = new Set(expandedBodies.value);
  if (next.has(commitId)) next.delete(commitId);
  else next.add(commitId);
  expandedBodies.value = next;
}

function isBodyOpen(commitId: string): boolean {
  return expandedBodies.value.has(commitId);
}

/** 点哈希复制完整 commit id：宿主有 copy 能力时优先用，兜底走浏览器剪贴板。 */
async function copyCommit(commitId: string): Promise<void> {
  if (!commitId) return;
  try {
    if (window.dbxPlugin?.copy) {
      await window.dbxPlugin.copy(commitId);
    } else {
      await navigator.clipboard.writeText(commitId);
    }
    copiedId.value = commitId;
    window.setTimeout(() => {
      if (copiedId.value === commitId) copiedId.value = "";
    }, 1500);
  } catch {
    // 复制失败不打断阅读，静默即可。
  }
}
</script>

<template>
  <section class="build-view">
    <header class="build-head">
      <div class="title-block">
        <button type="button" class="back" @click="emit('back')">← {{ t("backToJob") }}</button>
        <div class="title-row">
          <h1>#{{ build.number }}</h1>
          <span class="status-pill" :data-tone="statusMeta(build.status).tone">
            {{ t(statusMeta(build.status).key) }}
          </span>
        </div>
        <p class="full-name">{{ build.fullDisplayName || jobFullName }}</p>
        <p v-if="build.causes?.length" class="cause">
          {{ t("cause") }}: {{ build.causes.map((cause) => cause.shortDescription).join(" · ") }}
        </p>
      </div>
      <div class="actions">
        <RefreshBadge
          :mode="refresh.mode"
          :paused="refresh.paused"
          :in-flight="refresh.inFlight"
          :last-at="refresh.lastAt"
          :interval-ms="refresh.intervalMs"
        />
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
        <button
          v-if="build.building && !readOnly"
          type="button"
          class="danger"
          @click="emit('stop', build.number)"
        >
          {{ t("stopBuild") }}
        </button>
      </div>
    </header>

    <div class="meta-grid">
      <div class="meta">
        <span class="meta-label">{{ t("status") }}</span>
        <strong>{{ t(statusMeta(build.status).key) }}</strong>
      </div>
      <div class="meta">
        <span class="meta-label">{{ t("startedAt") }}</span>
        <strong>{{ formatTimestamp(build.timestamp, locale) }}</strong>
      </div>
      <div class="meta">
        <span class="meta-label">{{ t("duration") }}</span>
        <strong>{{ formatDuration(build.duration) }}</strong>
      </div>
      <div class="meta">
        <span class="meta-label">ETA</span>
        <strong>{{ formatDuration(build.estimatedDuration) }}</strong>
      </div>
    </div>

    <section v-if="build.changeSet?.length" class="panel">
      <div class="panel-head">
        <h2>{{ t("changes") }}</h2>
        <span class="count">{{ summaryText }}</span>
      </div>
      <table class="changes">
        <thead>
          <tr>
            <th class="col-commit">{{ t("commit") }}</th>
            <th class="col-time">{{ t("committedAt") }}</th>
            <th class="col-author">{{ t("author") }}</th>
            <th>{{ t("message") }}</th>
            <th class="col-files">{{ t("changedFiles") }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(item, index) in rows" :key="item.commitId || index">
            <td class="col-commit">
              <button
                type="button"
                class="commit-id"
                :title="t('copyCommit')"
                @click="copyCommit(item.commitId)"
              >
                <code>{{ shortCommitId(item.commitId) }}</code>
                <span v-if="copiedId === item.commitId" class="copied">{{ t("copied") }}</span>
              </button>
            </td>
            <td class="col-time">{{ formatChangeTime(item.timestamp ?? 0, locale) }}</td>
            <td class="col-author" :title="item.author">{{ item.author || "—" }}</td>
            <td class="message">
              <span class="subject">{{ item.message || "—" }}</span>
              <button
                v-if="hasBody(item)"
                type="button"
                class="toggle-body"
                @click="toggleBody(item.commitId)"
              >
                {{ isBodyOpen(item.commitId) ? t("hideFullMessage") : t("showFullMessage") }}
              </button>
              <pre v-if="hasBody(item) && isBodyOpen(item.commitId)" class="body">{{
                changeBody(item)
              }}</pre>
            </td>
            <td class="col-files">{{ item.fileCount || "—" }}</td>
          </tr>
        </tbody>
      </table>
      <div v-if="orderedChanges.length > CHANGE_PREVIEW_LIMIT" class="more">
        <button type="button" class="ghost" @click="expanded = !expanded">
          {{ expanded ? t("collapseAll") : t("expandAll", { count: orderedChanges.length }) }}
        </button>
      </div>
    </section>

    <section v-else class="panel">
      <div class="panel-head">
        <h2>{{ t("changes") }}</h2>
        <span class="count">{{ t("noChanges") }}</span>
      </div>
      <p class="empty-hint">{{ t("noChangesHint") }}</p>
    </section>

    <section v-if="build.artifacts?.length" class="panel">
      <div class="panel-head">
        <h2>{{ t("artifacts") }}</h2>
        <span class="count">{{ build.artifacts.length }}</span>
      </div>
      <ul class="artifacts">
        <li v-for="artifact in build.artifacts" :key="artifact.relativePath">
          <span>{{ artifact.fileName }}</span>
          <code>{{ artifact.relativePath }}</code>
        </li>
      </ul>
    </section>

    <ConsoleView :job="jobFullName" :number="build.number" />
  </section>
</template>

<style scoped>
.build-view {
  flex: 1;
  min-width: 0;
  overflow: auto;
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.build-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.back {
  border: 0;
  background: transparent;
  color: var(--muted);
  padding: 0;
  margin-bottom: 8px;
  font-size: 12px;
}

.back:hover {
  color: var(--accent);
}

.title-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

h1 {
  margin: 0;
  font-size: 22px;
}

.full-name {
  margin: 4px 0 0;
  color: var(--muted);
  font-size: 12px;
}

.cause {
  margin: 8px 0 0;
  color: var(--muted);
  font-size: 12px;
}

.danger {
  border: 1px solid color-mix(in srgb, var(--danger) 45%, transparent);
  border-radius: var(--radius-sm);
  padding: 8px 13px;
  background: transparent;
  color: var(--danger);
  font-weight: 600;
}

.ghost {
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  padding: 8px 13px;
  background: transparent;
  color: var(--muted);
  font-weight: 600;
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

.changes {
  width: 100%;
  table-layout: fixed;
  border-collapse: collapse;
  font-size: 12px;
}

.changes th {
  text-align: left;
  color: var(--muted);
  font-weight: 600;
  padding: 7px 8px;
  border-bottom: 1px solid var(--border);
}

.changes td {
  padding: 8px;
  border-bottom: 1px solid var(--border);
  vertical-align: top;
}

.col-commit {
  width: 116px;
}

.col-time {
  width: 104px;
  white-space: nowrap;
  color: var(--muted);
  font-variant-numeric: tabular-nums;
}

.col-author {
  width: 104px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.col-files {
  width: 56px;
  text-align: right;
  font-variant-numeric: tabular-nums;
  color: var(--muted);
}

.commit-id {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: 0;
  padding: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
}

.commit-id code {
  font-size: 11px;
  color: var(--text);
}

.commit-id:hover code {
  color: var(--accent);
}

.copied {
  color: var(--success);
  font-size: 11px;
}

.message {
  line-height: 1.5;
}

.subject {
  word-break: break-word;
}

.toggle-body {
  margin-left: 6px;
  border: 0;
  padding: 0;
  background: transparent;
  color: var(--accent);
  font-size: 11px;
  cursor: pointer;
}

.toggle-body:hover {
  text-decoration: underline;
}

.body {
  margin: 6px 0 0;
  color: var(--muted);
  font-family: inherit;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  word-break: break-word;
}

.more {
  margin-top: 10px;
  text-align: center;
}

.empty-hint {
  margin: 0;
  color: var(--muted);
  font-size: 12px;
}

.artifacts {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 12px;
}

.artifacts li {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 7px 9px;
  border-radius: var(--radius-sm);
  background: var(--panel-soft);
}

.artifacts code {
  color: var(--muted);
}
</style>
