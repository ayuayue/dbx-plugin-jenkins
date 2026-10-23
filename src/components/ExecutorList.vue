<script setup lang="ts">
import { computed } from "vue";
import { t } from "../i18n";
import {
  executorJobLabel,
  executorNodeLabel,
  executorProgress,
  executorState,
  executorSummary,
} from "../lib/executors";
import type { ExecutorInfo } from "../types";
import BuildProgressBar from "./BuildProgressBar.vue";

const props = defineProps<{
  rows: ExecutorInfo[];
  loading: boolean;
}>();

const emit = defineEmits<{ open: [row: ExecutorInfo] }>();

const summary = computed(() => executorSummary(props.rows));

function nodeLabel(row: ExecutorInfo): string {
  return executorNodeLabel(row.node, t("builtInNode"));
}

function stateLabel(row: ExecutorInfo): string {
  switch (executorState(row)) {
    case "idle":
      return t("executorIdle");
    case "pending":
      return t("executorPending");
    case "building":
      return t("statusBuilding");
    default:
      return t("executorSettling");
  }
}

/** 没有任务名的槽位（空闲 / 等待启动）点不动。 */
function openRow(row: ExecutorInfo): void {
  if (!row.jobName) return;
  emit("open", row);
}
</script>

<template>
  <section class="panel">
    <div class="panel-head">
      <h2>{{ t("buildExecutorStatus") }}</h2>
      <span class="count">{{ summary.busy }}/{{ summary.total }}</span>
    </div>

    <p v-if="loading" class="empty">{{ t("loading") }}</p>
    <p v-else-if="!rows.length" class="empty">{{ t("noExecutors") }}</p>
    <table v-else class="executors">
      <thead>
        <tr>
          <th>{{ t("nodeName") }}</th>
          <th>{{ t("executorNo") }}</th>
          <th>{{ t("jobs") }}</th>
          <th>{{ t("builds") }}</th>
          <th>{{ t("status") }}</th>
          <th>{{ t("progress") }}</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="row in rows"
          :key="`${row.node}-${row.index}`"
          :class="{
            clickable: !!row.jobName,
            idle: executorState(row) === 'idle',
            offline: row.nodeOffline,
          }"
          @click="openRow(row)"
        >
          <td class="node">
            {{ nodeLabel(row) }}
            <span v-if="row.nodeOffline" class="offline-tag">{{ t("offline") }}</span>
          </td>
          <td class="index">#{{ row.index }}</td>
          <td class="job">{{ row.jobName || "—" }}</td>
          <td class="build">{{ executorJobLabel(row) || "—" }}</td>
          <td>
            <span class="state" :data-state="executorState(row)">
              {{ stateLabel(row) }}
            </span>
          </td>
          <td class="progress-cell">
            <BuildProgressBar :progress="executorProgress(row)" />
          </td>
        </tr>
      </tbody>
    </table>
  </section>
</template>

<style scoped>
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
  font-variant-numeric: tabular-nums;
}

.empty {
  margin: 8px 0;
  color: var(--muted);
  font-size: 12px;
}

.executors {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}

.executors th {
  text-align: left;
  color: var(--muted);
  font-weight: 600;
  padding: 7px 8px;
  border-bottom: 1px solid var(--border);
}

.executors td {
  padding: 8px;
  border-bottom: 1px solid var(--border);
}

.executors tbody tr:last-child td {
  border-bottom: 0;
}

.executors tbody tr.clickable {
  cursor: pointer;
}

.executors tbody tr.clickable:hover {
  background: var(--panel-soft);
}

.executors tbody tr.idle td {
  color: var(--muted);
}

/* 构建中的槽位左侧标一条色带，和构建列表的「刚变化」高亮区分开 */
.executors tbody tr:not(.idle):not(.offline) td:first-child {
  box-shadow: inset 2px 0 0 var(--accent);
}

.node {
  font-weight: 600;
}

.offline-tag {
  margin-left: 6px;
  padding: 1px 6px;
  border-radius: 999px;
  background: color-mix(in srgb, var(--danger) 14%, transparent);
  color: var(--danger);
  font-size: 10px;
  font-weight: 600;
}

.index,
.build {
  color: var(--muted);
  font-variant-numeric: tabular-nums;
}

.build {
  white-space: nowrap;
}

.job {
  max-width: 280px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.state {
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--panel-soft);
  color: var(--muted);
  font-size: 10px;
  font-weight: 700;
}

/* 色相交给底色 + 描边，文字统一用正文色：深色主题下纯 accent 文字对比不足 */
.state[data-state="building"] {
  background: var(--accent-soft);
  color: var(--text);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--accent) 35%, transparent);
}

.state[data-state="pending"] {
  background: color-mix(in srgb, var(--warning) 16%, transparent);
  color: var(--text);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--warning) 40%, transparent);
}

.progress-cell {
  width: 132px;
}
</style>
